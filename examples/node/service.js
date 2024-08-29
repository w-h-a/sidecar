const express = require('express');
const URL = require('url').URL;

const app = express();

app.use(express.json());

const port = 3000;

const actionsUrl = 'http://node-action:3501';

app.post('/neworder-node', async (req, res) => {
  const data = req.body.data;

  const orderId = data.orderId;

  console.log('got a new order: ' + orderId);

  const records = [
    {
      key: orderId.toString(),
      value: data,
    },
  ];

  const url = new URL(`/state/orders`, actionsUrl);

  const rsp = await fetch(url.href, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
    },
    body: JSON.stringify(records),
  });

  if (rsp.ok) {
    console.log('successfully persisted state');
  } else {
    console.log('failed to persist state');
  }

  res.json({});
});

app.get('/order', async (req, res) => {
  const url = new URL(`/state/orders`, actionsUrl);

  const rsp = await fetch(url.href, {
    method: 'GET',
    headers: {
      'content-type': 'application/json',
    },
  });

  const body = await rsp.json();

  console.log(`we received this from actions: ${JSON.stringify(body)}`);

  res.json(body);
});

app.get('/order/:id', async (req, res) => {
  const id = req.params['id'];

  const url = new URL(`/state/orders/${id}`, actionsUrl);

  const rsp = await fetch(url.href, {
    method: 'GET',
    headers: {
      'content-type': 'application/json',
    },
  });

  const body = await rsp.json();

  console.log(`we received this from actions: ${JSON.stringify(body)}`);

  res.json(body);
});

app.delete('/order/:id', async (req, res) => {
  const id = req.params['id'];

  const url = new URL(`/state/orders/${id}`, actionsUrl);

  await fetch(url.href, {
    method: 'DELETE',
    headers: {
      'content-type': 'application/json',
    },
  });

  res.json({});
});

app.listen(port, () => {
  console.log(`node is listening on port ${port}`);
});
