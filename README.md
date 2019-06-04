一个低性能端口转发实现

## 使用

要求配置文件和程序放在一起，且名字必须是 `config.json`, 配置文件格式为

```typescript
interface Config {
  proxy: {
    [localPort: string]: string;
  }
  APIPort: string;
}
```

其中 `APIPort` 为 api 监听的端口。

示例参见 [示例配置](config.json)

## web 控制台

提供了一个[简陋的web](https://muzea.github.io/portfwd/web/)。

## api

- 均要求数据为 `application/json`
- 类型描述为typescript

### ping

`GET /ping`

request

无

response
```typescript
interface Resp {
  message: 'pong'
}
```

### add

`POST /proxy`

request

```typescript
interface Req {
  local: string // 比如 "10086"
  targrt: string //比如 "127.0.0.1:10010"
}
```

response
```typescript
interface Resp {
  message: 'done'
}
```

### update

`PATCH /proxy/:localPort`

request

```typescript
interface Req {
  targrt: string //比如 "127.0.0.1:10010"
}
```

response
```typescript
interface Resp {
  message: 'done'
}
```

### delete

`DELETE /proxy/:localPort`

request

无

response
```typescript
interface Resp {
  message: 'done'
}
```

### proxy config

`GET /proxy`

request

无

response
```typescript
type Resp = {
    [k in keyof ProxyPool]: string
}
```

### proxy item

`GET /proxy/:localPort`

request

无

response
```typescript
interface Resp {
  local: string
  target: string
}
```

