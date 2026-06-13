# rollops-plugin-unleash

A [Rollops](https://github.com/klarlabs-studio/rollops) feature-flag provider
plugin backed by [Unleash](https://www.getunleash.io/). It drives an Unleash
feature toggle's enabled state and the rollout percentage of its
`flexibleRollout` (gradual rollout) strategy to track a Rollops canary in
lockstep — as a rollout steps 10% → 50% → 100%, the flag follows.

## How it works

Rollops calls the plugin per progressive step (and/or on promote) with the flag
name, the target environment, and the current traffic percentage. The plugin:

1. Looks up the feature in the configured project.
2. Sets the `flexibleRollout` strategy's `rollout` parameter to the percentage
   for that environment, creating the strategy if the environment lacks one.
3. Enables or disables the feature in the environment.

## Configuration

The plugin reads its credentials from its own environment — never from the
Rollops target spec, which carries only the flag name, environment, and
percentage:

| Env var           | Required | Default | Description                              |
|-------------------|----------|---------|------------------------------------------|
| `UNLEASH_API_URL` | yes      | —       | Base URL, e.g. `https://unleash.example.com` |
| `UNLEASH_TOKEN`   | yes      | —       | Admin API token (`Authorization: <token>`) |
| `UNLEASH_PROJECT` | no       | `default` | Unleash project id                     |

## Install

Via the Rollops marketplace:

```sh
rollops plugin install unleash
```

Or build and pin manually:

```sh
make build
make checksum   # prints the sha256 to pin
```

Then wire it into a rollout spec:

```yaml
featureFlags:
  plugin: ~/.rollops/plugins/unleash
  sha256: <pin>
  flag: checkout
  environment: production
  when: both        # step | promote | both
```

## License

MIT
