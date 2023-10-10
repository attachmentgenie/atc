Kind           = "service-resolver"
Name           = "web"
ConnectTimeout = "15s"
Failover = {
  "*" = {
    Datacenters = ["dc2"]
  }
}

