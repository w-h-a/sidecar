const express = require('express');
const URL = require('url').URL;

const app = express();

app.use(express.json());

const port = 3000;

const actionsUrl = 'http://node-action:3501';

app.post('/neworder', async (req, res) => {
  const data = req.body.data;

  const url = new URL(`/publish`, actionsUrl);

  await fetch(url.href, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
    },
    body: JSON.stringify({
      eventName: 'neworder-queue',
      data: data,
      to: ['neworder-queue'],
    }),
  });

  res.json({});
});

app.post('/neworder-queue', async (req, res) => {
  const data = req.body.data;

  const orderId = data.orderId;

  console.log('got a new order: ' + orderId);

  res.json({
    state: {
      storeId: 'orders',
      records: [
        {
          key: orderId,
          value: data,
        },
      ],
    },
  });
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

app.listen(port, () => {
  console.log(`node is listening on port ${port}`);
});
