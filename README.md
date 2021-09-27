### Missions service

Depends on redis

Uses redis as storage

No authentification or authorization. Please use it behind the API gateway with auth

#### Environment variables

  - WIPEKEY - Salt to generate requests for wipe user progress
  - REWARD_HOOK_URL - URL to POST when reward needs to be allocated

#### Wipe key generation

SHA256({EPOCH}{USERID}{WIPEKEY})


#### Claim mission


Claim mission sends request to REST API via 

POST REWARD_HOOK_URL

```
{
	userid:"",
	reward:"",
	amount:0,
}
```
