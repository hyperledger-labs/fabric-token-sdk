# Drivers

The Token SDK comes equipped with two driver implementations:
- [`FabToken`](./fabtoken.md): This is a simple implementation of the Driver API that does not support privacy.
- [`ZKAT DLog`](./zkat-dlog.md): This driver supports privacy via Zero Knowledge. We follow
  a simplified version of the blueprint described in the paper
  [`Privacy-preserving auditable token payments in a permissioned blockchain system`]('https://eprint.iacr.org/2019/1058.pdf')
  by Androulaki et al.