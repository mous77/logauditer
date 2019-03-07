# logauditer

* 依赖mongodb存储

* 远程日志格式模式

| 文件      |      表达式    |  
|----------|:-------------:|
|10.10.2.104_2018-12-04_DianXin-route.log |  (\\d+.\\d+.\\d+.\\d+_\\d+-route.log)|
|10.10.4.32_2018-12-04_sshd.log  |      (\\d+.\\d+.\\d+.\\d+_\\d+_sshd.log) |

* server编译

```shell
# 启动服务需依赖mongodb
# dburl: "127.0.0.1:27017"
cd $GOPATH/src/cmd/server
make
```

* client编译

```shell
cd $GOPATH/src/cmd/client
make
```

* 测试用例 client 交互

```javascript
set rule
 `{
	"dir": "/monitdir",  -- 目录
	"device": "x86server(const)",    -- 设备类型,可随便定义
	"systemType": "server",          -- 系统类型:[server/switch/app]
	"filePattern": "(\\d+.\\d+.\\d+.\\d+.*.log)",   -- 文件名表达式
	"logDate": "(\\d+-\\d+-\\d+-\\d+)",             -- 日志时间(由日志文件声明)
	"host": "(\\d+.\\d+.\\d+.\\d+)",                -- 主机名/ip
	"linePrefixPattern": "(\\w+  \\d \\d+:\\d+:\\d+ \\w+ \\w+: \\w+     \\w+\\/\\d+        \\d+-\\d+-\\d+ \\d+:\\d+ \\(\\d+.\\d+.\\d+.\\d+\\) \\[\\d+\\]:)",  -- 行头,一般叫日志固定的格式
	"columnPattern": {         -- 日志内容列表达多
		"UserName": 6,         -- 用户名,例如系统日志里的操作用户
		"IpAddr": "(\\(\\d+.\\d+.\\d+.\\d+\\))", -- 操作者的IP地址
		"State": "(\\[\\d\\])",   -- 操作的执行状态
		"DateTime": "(\\w+\\  \\d\\ \\d+:\\d+:\\d+)" -- 日志中内容操作时间
	}
 }`;
 ```

* 测试&提交&启动规则

```javascript
test `Jan  1 14:21:09 mongo521 root: root     pts/0        2019-01-07 14:19 (10.10.3.133) [432662]: scp -r mongodb-linux-x86_64-rhel70-4.0.2.tgz root@10.10.3.41:/root [1]` with rule;

commit rule;

start rule;
```

* 测试写入文件

```shell
echo "Jan  1 14:21:09 mongo521 root: root     pts/0        2019-01-07 14:19 (10.10.3.133) [432662]: scp -r mongodb-linux-x86_64-rhel70-4.0.2.tgz root@10.10.3.41:/root [1]" >> 10.10.2.104_2018-12-04_RawStore.log
echo "Jan  2 14:21:09 mongo521 root: root     pts/0        2019-01-07 14:19 (10.10.3.133) [432662]: scp -r mongodb-linux-x86_64-rhel70-4.0.2.tgz root@10.10.3.41:/root [1]" >> 10.10.2.104_2018-12-04_RawStore1.log
echo "Jan  3 14:21:09 mongo521 root: root     pts/0        2019-01-07 14:19 (10.10.3.133) [432662]: scp -r mongodb-linux-x86_64-rhel70-4.0.2.tgz root@10.10.3.41:/root [1]" >> 10.10.2.104_2018-12-04_RawStore2.log
echo "Jan  4 14:21:09 mongo521 root: root     pts/0        2019-01-07 14:19 (10.10.3.133) [432662]: scp -r mongodb-linux-x86_64-rhel70-4.0.2.tgz root@10.10.3.41:/root [1]" >> 10.10.2.104_2018-12-04_RawStore3.log
echo "Jan  5 14:21:09 mongo521 root: root     pts/0        2019-01-07 14:19 (10.10.3.133) [432662]: scp -r mongodb-linux-x86_64-rhel70-4.0.2.tgz root@10.10.3.41:/root [1]" >> 10.10.2.104_2018-12-04_RawStore4.log
echo "Jan  6 14:21:09 mongo521 root: root     pts/0        2019-01-07 14:19 (10.10.3.133) [432662]: scp -r mongodb-linux-x86_64-rhel70-4.0.2.tgz root@10.10.3.41:/root [1]" >> 10.10.2.104_2018-12-04_RawStore5.log
```
