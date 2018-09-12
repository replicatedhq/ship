FROM nginx:1.11.8
ARG dist
COPY ${dist} /usr/share/nginx/html
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
