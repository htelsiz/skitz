# Tailscale

`tailscale status` show connected devices and IPs ^run
`tailscale ip` show your Tailscale IPs ^run
`tailscale ping {{host}}` ping a device on your tailnet ^run:host
`tailscale netcheck` check network connectivity and latency ^run
`tailscale exit-node list` list available exit nodes ^run
`sudo tailscale set --exit-node={{node}}` set exit node ^run:node
`sudo tailscale set --exit-node=` clear exit node ^run
`sudo tailscale set --exit-node={{node}} --exit-node-allow-lan-access=true` exit node with LAN access ^run:node
`sudo tailscale up` connect to tailnet ^run
`sudo tailscale down` disconnect from tailnet ^run
`tailscale whois {{ip}}` look up a device by IP ^run:ip
`tailscale dns status` show DNS configuration ^run
`tailscale file send {{file}} {{host}}:` send file to device ^run:file,host
`tailscale file get {{directory}}` receive pending files ^run:directory
`tailscale cert {{domain}}` get TLS cert for a domain ^run:domain
`tailscale ssh {{user}}@{{host}}` SSH over Tailscale ^run:user,host
`sudo tailscale set --advertise-exit-node` advertise as exit node ^run
`sudo tailscale set --advertise-routes={{cidr}}` advertise subnet routes ^run:cidr
`tailscale lock status` check tailnet lock status ^run
`tailscale bugreport` generate debug info for support ^run
