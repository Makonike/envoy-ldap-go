# envoy-ldap-go

## Start

Download [glauth](https://github.com/glauth/glauth/releases), and change its [sample config file](https://github.com/glauth/glauth/blob/master/v2/sample-simple.cfg). 

sample.yaml

```yaml
[ldap]
  enabled = true
  # run on a non privileged port
  listen = "192.168.64.1:3893" # 192.168.64.1 is your local network IP. Please synchronize it with the envoy.yaml file.
```

Then, start it.

```bash
./glauth -c sample.yaml
```

Start your App.

```bash
make build
```

```bash
make run 
```

## Test

```bash
make test
```

