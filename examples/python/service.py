import time
import requests

actions_url = "http://python-action:3501/publish"

n = 0

while True:
  n += 1

  message = { "eventName": "neworder", "data": { "orderId": n }, "to": ["node"] }

  try:
    response = requests.post(actions_url, json=message)
  except Exception as e:
      print(e)

  time.sleep(10)
  