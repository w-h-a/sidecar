const express = require('express');
const URL = require('url').URL;

const app = express();

app.use(express.json());

const port = 3000;

const sidecarUrl = 'http://localhost:3501';

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

  const url = new URL(`/state/orders`, sidecarUrl);

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

app.post('/b-node', async (req, res) => {
  const data = req.body.data;

  console.log(`b: ${JSON.stringify(data)}`);

  res.json({});
});

app.get('/order', async (req, res) => {
  const url = new URL(`/state/orders`, sidecarUrl);

  const rsp = await fetch(url.href, {
    method: 'GET',
    headers: {
      'content-type': 'application/json',
    },
  });

  const body = await rsp.json();

  console.log(`we received this from sidecar: ${JSON.stringify(body)}`);

  res.json(body);
});

app.get('/order/:id', async (req, res) => {
  const id = req.params['id'];

  const url = new URL(`/state/orders/${id}`, sidecarUrl);

  const rsp = await fetch(url.href, {
    method: 'GET',
    headers: {
      'content-type': 'application/json',
    },
  });

  const body = await rsp.json();

  console.log(`we received this from sidecar: ${JSON.stringify(body)}`);

  if (rsp.status === 404) {
    res.status(404).json(body);
    return;
  }

  res.json(body);
});

app.delete('/order/:id', async (req, res) => {
  const id = req.params['id'];

  const url = new URL(`/state/orders/${id}`, sidecarUrl);

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
