# atc

Like with actual airports we sometimes need a process that controls what should happen with requests for consul services. 
manually setting up failover and redirect config can be quite laborious. ATC can help by creating those for you by utilising 
consul watchers.

## install

### os x
    brew tap attachmentgenie/atc https://github.com/attachmentgenie/atc.git
    brew install atc
