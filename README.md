# atc

Like with actual airports we sometimes need a process that controls what should happen with requests for (failing) consul services.
manually setting up failover and redirect consul service-resolver config can be quite laborious. ATC can help make live that little bit easier by creating those for us.

## install

### os x

    brew tap attachmentgenie/tap
    brew install attachmentgenie/tap/atc
