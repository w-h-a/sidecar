FROM python:3.7.1-alpine3.8
WORKDIR /service
COPY ./examples/python .
RUN pip install requests
ENTRYPOINT ["python"]
CMD ["service.py"]