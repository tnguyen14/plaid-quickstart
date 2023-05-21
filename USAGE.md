## Tri's usage

If an access token already exists, with an Item ID, set their values in `.env`

```
ACCESS_TOKEN=access-development-xxx
ITEM_ID=yyy
```
and restart the node server.

Upon loading the frontend, "Launch Link", it will authenticate with the access token.

From what I could tell, if an Item requires re-login, it will issue a new Item ID. If the Item ID is still good, it will just load that Item ID.
