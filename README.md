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
TODO
```

I've also included two example `nftables` files. The first `XXXX` is an example usage
of the defintions file. The second `YYYY` is an example usage for hotswapping the
contents of the bogons sets, without reloading the whole ruleset.

[fullbogons-ref]: https://team-cymru.com/community-services/bogon-reference/
