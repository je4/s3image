package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/je4/s3image/v2/pkg/filesystem"
	"github.com/je4/s3image/v2/pkg/media"
	dcert "github.com/je4/utils/v2/pkg/cert"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type Server struct {
	service        string
	host, port     string
	name, password string
	srv            *http.Server
	log            *logging.Logger
	accessLog      io.Writer
	templates      map[string]*template.Template
	fs             filesystem.FileSystem
}

func NewServer(service, addr, name, password string, log *logging.Logger, accessLog io.Writer, fs filesystem.FileSystem) (*Server, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot split address %s", addr)
	}

	srv := &Server{
		service:   service,
		host:      host,
		port:      port,
		name:      name,
		password:  password,
		log:       log,
		accessLog: accessLog,
		templates: map[string]*template.Template{},
		fs:        fs,
	}

	return srv, srv.InitTemplates()
}

func (s *Server) InitTemplates() error {
	for key, val := range templateFiles {
		tpl, err := template.ParseFS(templateFS, val)
		if err != nil {
			return errors.Wrapf(err, "cannot parse template %s: %s", key, val)
		}
		s.templates[key] = tpl
	}
	return nil
}
func (s *Server) IndexHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	path := strings.Trim(vars["path"], "/")

	parts := strings.SplitN(path, "/", 2)
	if parts == nil {
		s.log.Infof("invalid path %s", path)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("invalid path %s", path)))
		return
	}

	de, err := s.fs.FileList(parts[0], parts[1])
	if err != nil {
		if parts == nil {
			s.log.Infof("cannot read folder %s: %v", path, err)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(fmt.Sprintf("cannot read folder %s: %v", path, err)))
			return
		}
	}

	tpl := s.templates["index"]
	if err := tpl.Execute(w, struct {
		Path    string
		Entries []os.DirEntry
	}{path, de}); err != nil {
		s.log.Errorf("error executing index template: %v", err)
	}
}

func (s *Server) MasterHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	path := strings.TrimPrefix(vars["path"], "/")

	parts := strings.SplitN(path, "/", 2)
	if parts == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("cannot open file %s", path)))
		return
	}

	r, contentType, err := s.fs.FileOpenRead(parts[0], parts[1], filesystem.FileGetOptions{})
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("cannot open file %s", path)))
		return
	}
	w.Header().Add("Content-type", contentType)
	if _, err := io.Copy(w, r); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("cannot read file %s", path)))
		return
	}
}

func (s *Server) ThumbHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	path := strings.TrimPrefix(vars["path"], "/")

	parts := strings.SplitN(path, "/", 2)
	if parts == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("cannot open file %s", path)))
		return
	}

	r, _, err := s.fs.FileOpenRead(parts[0], parts[1], filesystem.FileGetOptions{})
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("cannot open file %s", path)))
		return
	}

	var image media.ImageType
	image, err = media.NewImageMagickV3(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("read image %s", path)))
		return
	}
	defer image.Close()

	imageOptions := &media.ImageOptions{
		Width:             359,
		Height:            225,
		ActionType:        media.ResizeActionTypeKeep,
		TargetFormat:      "JPEG",
		OverlayCollection: "",
		OverlaySignature:  "",
		BackgroundColor:   "",
	}
	if err := image.Resize(imageOptions); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("resize image %s", path)))
		return
	}
	reader, _, err := image.StoreImage("JPEG")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("store image %s", path)))
		return
	}
	w.Header().Add("Content-type", "image/jpg")
	if _, err := io.Copy(w, reader); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("output image %s", path)))
		return
	}
}

var thumbPath = regexp.MustCompile("^(?P<path>.+)/thumb$")
var masterPath = regexp.MustCompile("^(?P<path>.+)/master$")
var indexPath = regexp.MustCompile("^(?P<path>.+)$")

func (s *Server) ListenAndServe(cert, key string) (err error) {
	router := mux.NewRouter()

	router.MatcherFunc(func(request *http.Request, match *mux.RouteMatch) bool {
		matches := indexPath.FindStringSubmatch(request.URL.Path)
		if matches == nil {
			return false
		}
		match.Vars = map[string]string{}
		for i, name := range indexPath.SubexpNames() {
			if name == "" {
				continue
			}
			if strings.HasSuffix(matches[i], "/thumb") {
				return false
			}
			if strings.HasSuffix(matches[i], "/master") {
				return false
			}
			match.Vars[name] = matches[i]
		}
		return true
	}).Methods("GET", "HEAD").HandlerFunc(s.IndexHandler)

	router.MatcherFunc(func(request *http.Request, match *mux.RouteMatch) bool {
		matches := thumbPath.FindStringSubmatch(request.URL.Path)
		if matches == nil {
			return false
		}
		match.Vars = map[string]string{}
		for i, name := range thumbPath.SubexpNames() {
			if name == "" {
				continue
			}
			match.Vars[name] = matches[i]
		}
		return true
	}).Methods("GET", "HEAD").HandlerFunc(s.ThumbHandler)

	router.MatcherFunc(func(request *http.Request, match *mux.RouteMatch) bool {
		matches := masterPath.FindStringSubmatch(request.URL.Path)
		if matches == nil {
			return false
		}
		match.Vars = map[string]string{}
		for i, name := range masterPath.SubexpNames() {
			if name == "" {
				continue
			}
			match.Vars[name] = matches[i]
		}
		return true
	}).Methods("GET", "HEAD").HandlerFunc(s.MasterHandler)

	loggedRouter := handlers.CombinedLoggingHandler(s.accessLog, handlers.ProxyHeaders(router))
	addr := net.JoinHostPort(s.host, s.port)
	s.srv = &http.Server{
		Handler: loggedRouter,
		Addr:    addr,
	}

	if cert == "auto" || key == "auto" {
		s.log.Info("generating new certificate")
		cert, err := dcert.DefaultCertificate()
		if err != nil {
			return errors.Wrap(err, "cannot generate default certificate")
		}
		s.srv.TLSConfig = &tls.Config{Certificates: []tls.Certificate{*cert}}
		return s.srv.ListenAndServeTLS("", "")
	} else if cert != "" && key != "" {
		return s.srv.ListenAndServeTLS(cert, key)
	} else {
		return s.srv.ListenAndServe()
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
