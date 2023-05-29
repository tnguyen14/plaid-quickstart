## Tri's usage

```sh
cd node
npm start  # npm ci first on first time run
# or
npx nodemon index.js

---
cd frontend
npm start  # npm i first on first time run
```
Go to http://localhost:3000


### Existing item
If an access token already exists, with an Item ID, set their values in `.env`

```
ACCESS_TOKEN=access-development-xxx
ITEM_ID=yyy
```

Update `node/index.js` for endpoint `/api/create_link_token` to comment out `products` in the payload.

Restart the node server.

Upon loading the frontend, "Launch Link", it will authenticate with the access token.

From what I could tell, if an Item requires re-login, it will issue a new Item ID. If the Item ID is still good, it will just load that Item ID.

### New item

Uncomment out the `products` attribute in `node/index.js` under `/api/create_link_token` endpoint, if it was commented out.
