# `fullbogons-nftables-gen`

The tool exists to create a definition file for `nftables` from the fullbogons
files from [Team Cymru][fullbogons-ref]. It can be used to filter out bogons
from incoming traffic with `nftables`.

There is probabily little advantage to this, but I wanted to play around with
named sets and so forth.

**This is duct tape programming max.** There are no tests and there is no plans
to add any. It's for a very specific usecase I had and it's not really going
to expand beyond this.

## Usage

The intended usage is to be used in `cron` or `systemd.timers`, to generate a
definition file. The name of this file is the only argument to the program.
The file will be atomically replaced, so there should not be an instance of
a half written file, as long as `rename` is implemented atomically.

## Extras

To make usage easier, I've included a `systemd.timer` file that will execute this
once every 4 hours roughly. This is how you can install it:

```
sudo install fullbogons-nftables-gen /usr/local/bin
sudo install -m644 contrib/systemd/fullbogons-nftables-gen.* /etc/systemd/system
sudo install -Dm644 contrib/nftables/example-refresh-set.nft /etc/nftables.d/refresh-fullbogons.nft

# This will fail, and that is ok, it is just to get the inital definitions file.
sudo systemctl start fullbogons-nftables-gen.service

# Then update /etc/nftables.conf with something akin to the content of
# contrib/nftables/example-ruleset.nft

# Then enable the timer
sudo systemctl enable --now fullbogons-nftables-gen.timer
```

I've also included two example `nftables` files. The first `contrib/nftables/example-ruleset.nft`
is an example usage of the defintions file. The second 
`contrib/nftables/example-refresh-set.nft` is an example usage for hotswapping the
contents of the bogons sets, without reloading the whole ruleset.

[fullbogons-ref]: https://team-cymru.com/community-services/bogon-reference/
