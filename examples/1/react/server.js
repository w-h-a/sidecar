const express = require('express');
const URL = require('url').URL;
const path = require('path');

const port = 8080;

const sidecarUrl = 'http://localhost:3501';

const app = express();
app.use(express.json());
app.use(express.static(path.join(__dirname, 'client/build')));

app.post('/publish', async (req, res) => {
  console.log(`publishing: ${JSON.stringify(req.body)}`);

  const url = new URL(`/publish`, sidecarUrl);

  await fetch(url.href, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
    },
    body: JSON.stringify(req.body),
  });

  res.json({});
});

app.get('*', async (req, res) => {
  res.sendFile(path.join(__dirname, 'client/build', 'index.html'));
});

app.listen(port, () => {
  console.log(`listening on port ${port}`);
});
