# BountyTracker
赏金目标监控

#### 安装
```shell
   go install github.com/baiqll/bountytr@latest
```

#### 优化
```
   1.0 使用的是 github.com/arkadiyt/bounty-targets-data 提供的数据，2.0 是自己写的数据抓取逻辑。
   优化域名检测正则 ，去除dns域名检测
   优化网络请求

```
#### 技术债

   bugcrowd 并发数超过15 会被限制
   intigriti 并发数超过60 会被限制
   hackerone 不限制并发数

   优化后 ：
      bugcrowd 最大用时1分钟
      intigriti 最大用时1分钟
      hackerone 最大用时1分钟


#### 参考
* https://github.com/arkadiyt/bounty-targets-data
