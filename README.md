# tidb-keyvisual
Visualization of the access mode of the key in the tidb cluster.

# Run keyvisual
required: `go1.13`

build:
```
go build . && ./keyvisual --pd=http://127.0.0.1:2379 --tidb=http://127.0.0.1:10080
```

open [http://localhost](http://localhost) in Browser. (If the page does't load, please wait for a while to fetch data.)


# arguments

```
-I duration
    	Interval to collect metrics (default 1m0s)
  -N int
    	Max Bucket number in the histogram (default 256)
  -addr string
    	Listening address (default "0.0.0.0:8000")
  -no-sys
    	Ignore system database (default true)
  -pd string
    	PD address (default "http://127.0.0.1:2379")
  -tidb string
    	TiDB Address (default "http://127.0.0.1:10080")

```
