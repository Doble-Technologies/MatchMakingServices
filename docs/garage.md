# Startup Scripes for Garage
```zsh
/garage layout assign   --zone dc1   --capacity 100G   NODE_ID
/garage layout show
/garage layout apply --version 1
/garage bucket create matchmaking
/garage key create mm-access-key
/garage bucket allow   --read --write   matchmaking   --key mm-access-key
/garage key info mm-access-key
```
