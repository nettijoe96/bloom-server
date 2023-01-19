# SPV & Bloom Filters
**See Bloom filter [package](https://github.com/nettijoe96/bloom)!**

This is a simulation of one aspect of SPV nodes. SPV nodes send a bloom to a full node and the full node responds with all transactions that match the bloom. Instead of transactions, this code uses simple string messages.

## How to run
`$ git clone https://github.com/nettijoe96/spv-bloom.git`
### Manual
```
$ cd spv-bloom
$ go run .
```
### Docker
```
$ cd spv-bloom
$ docker build -t spv-bloom .
$ docker run -dp 8080:8080 --rm spv-bloom
```

## API spec

After running, you can see the swagger docs in browser: http://127.0.0.1:8080/docs

## Example
```
POST 127.0.0.1:8080/publish
request:
{
    "messages" = [
        "hello",
        "world",
        "another message"
    ]
}
```

```
POST 127.0.0.1:8080/bloom-request
request:
{
    "bloom": {
        "filter": "00000000000000000001000001008000010000000000000100000000000000000000000000000001000000000000000000000100000000000000000000000040",
        "k": 4
    }
}

response:
{
    "messages":[
        "hello"
        "world"
    ]
}
```

## TODO
1. Database to store messages on server side
