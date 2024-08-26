import time
import requests

actions_url = "http://python-action:3501/publish"

while True:
  message = { "eventName": "neworder", "data": { "orderId": "777" }, "to": ["node"] }

  try:
    response = requests.post(actions_url, json=message)
  except Exception as e:
      print(e)

  time.sleep(10)
  