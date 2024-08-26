FROM node:20.11.1-alpine
WORKDIR /service
COPY ./examples/node .
RUN npm install
EXPOSE 3000
CMD [ "node", "service.js" ]