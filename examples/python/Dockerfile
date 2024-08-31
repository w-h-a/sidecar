FROM python:3.7.1-alpine3.8
WORKDIR /service
COPY ./examples/python .
RUN pip install flask
EXPOSE 4000
CMD ["python", "service.py"]