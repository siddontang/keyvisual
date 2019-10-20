# TiDB KeyVisual

Visualization of the access mode of the key in the tidb cluster.

## Run keyvisual

Required: `go1.13`.

Build and run:
```
go build .  
./keyvisual --pd=http://127.0.0.1:2379 --tidb=http://127.0.0.1:10080
```

Open [http://localhost:8000](http://localhost:8000) in Browser.
