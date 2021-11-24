FROM alpine:3

WORKDIR /app
# Now just add the binary
ADD s3image /app
ENTRYPOINT ["./s3image"]
EXPOSE 3000
