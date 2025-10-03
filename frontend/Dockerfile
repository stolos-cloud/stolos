FROM node:20-alpine AS build-stage
WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .
RUN npm run build

FROM nginx:stable-alpine AS prod-stage

COPY --from=build-stage /app/dist /usr/share/nginx/html
COPY ./nginx/default.conf.template /etc/nginx/templates/default.conf.template

EXPOSE 80

CMD ["sh", "-c", "envsubst '${API_URL}' < /etc/nginx/templates/default.conf.template > /etc/nginx/conf.d/default.conf && nginx -g 'daemon off;'"]