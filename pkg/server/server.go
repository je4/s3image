package server

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/je4/s3image/v2/pkg/filesystem"
	"github.com/je4/s3image/v2/pkg/media"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"html/template"
	"io"
	"net"
	"net/http"
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

func (s *Server) ThumbHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	path := vars["path"]

	parts := strings.SplitN(path, "/", 2)

	r, err := s.fs.FileOpenRead(parts[0], parts[1], filesystem.FileGetOptions{})
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

	if err := image.Resize(&media.ImageOptions{
		Width:             120,
		Height:            120,
		ActionType:        "resize",
		TargetFormat:      "PNG",
		OverlayCollection: "",
		OverlaySignature:  "",
		BackgroundColor:   "",
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("resize image %s", path)))
		return
	}
	reader, meta, err := image.StoreImage("PNG")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("store image %s", path)))
		return
	}
	w.Header().Add("Content-type", meta.Mimetype)
	if _, err := io.Copy(w, reader); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("output image %s", path)))
		return
	}
}

var path = regexp.MustCompile("^?P<path>(.+)/thumb$")

func (s *Server) ListenAndServe(cert, key string) (err error) {
	router := mux.NewRouter()

	router.MatcherFunc(func(request *http.Request, match *mux.RouteMatch) bool {
		matches := path.FindStringSubmatch(request.URL.Path)
		if matches == nil {
			return false
		}
		match.Vars = map[string]string{}
		for i, name := range path.SubexpNames() {
			if name == "" {
				continue
			}
			match.Vars[name] = matches[i]
		}
		return true
	}).Methods("GET", "HEAD").HandlerFunc(s.ThumbHandler)

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
