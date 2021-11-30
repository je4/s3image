FROM cr.gitlab.fhnw.ch/hgk-dima/containers/goimagemagick:latest
WORKDIR /app
# Now just add the binary
ADD s3image /app
ENTRYPOINT ["./s3image"]
EXPOSE 3000