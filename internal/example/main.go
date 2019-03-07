package main

import (
	"encoding/json"
	"fmt"
	"logauditer/internal"
	"os"
	//	"time"
)

var textSet = []string{

	//prefix pattern =
	//`(\w+  \d \d+:\d+:\d+ \w+ \w+: \w+     \w+\/\d+        \d+-\d+-\d+ \d+:\d+ \(\d+.\d+.\d+.\d+\) \[\d+\]:)`
	`Jan  7 14:21:09 mongo521 root: root     pts/0        2019-01-07 14:19 (10.10.3.133) [432662]: scp -r mongodb-linux-x86_64-rhel70-4.0.2.tgz root@10.10.3.41:/root [1]`,

	//`(\w+  \d \d+:\d+:\d+ \w+ \d+\: \*\w+  \d \d+:\d+:\d+\.\d+: \%\w+-\d+-\w+:)`
	`Jan  7 01:20:40 bogon 1741: *Jan  7 01:21:44.906: %SYS-5-CONFIG_I: Configured from console by yicheng on vty8 (10.10.10.10)`,

	//
	`Jan  7 09:40:33 2019 DianXin-route %%10SHELL/5/SHELL_LOGIN: -DevIP=10.10.2.104; yicheng logged in from 10.10.10.10.`,

	//
	`Jan  7 10:06:46 localhost sshd[18902]: pam_unix(sshd:session): session opened for user root by (uid=0)`,
}

func main() {

	logParts := internal.NewLogParts()

	rule1 := `{"dir": "/Users/xiaopengdeng/Documents/workspace/go-project/src/logauditer/cmd/monitdir", "device" : "server(const)", "systemType" : "server", "filePattern": "(\\d+.\\d+.\\d+.\\d+.*.log)", "logDate": "(\\d+-\\d+-\\d+-\\d+)", "host": "(\\d+.\\d+.\\d+.\\d+)", "linePrefixPattern": "(\\w+  \\d \\d+:\\d+:\\d+ \\w+ \\w+: \\w+     \\w+\\/\\d+        \\d+-\\d+-\\d+ \\d+:\\d+ \\(\\d+.\\d+.\\d+.\\d+\\) \\[\\d+\\]:)","columnPattern": {"UserName": 6,"IpAddr": "(\\(\\d+.\\d+.\\d+.\\d+\\))","State": "(\\[\\d\\])","DateTime": "(\\w+\\  \\d\\ \\d+:\\d+:\\d+)"}}`
	_ = rule1
	rule2 := `{"dir": "/tmp","systemType": "server","device": "x86-sever","filePattern": "(\\w+_\\w+.log)","linePrefixPattern": "(\\w+  \\d \\d+:\\d+:\\d+ \\w+ \\w+: \\w+     \\w+\\/\\d+        \\d+-\\d+-\\d+ \\d+:\\d+ \\(\\d+.\\d+.\\d+.\\d+\\) \\[\\d+\\]:)","columnPattern": {"UserName": 6,"IpAddr": "(\\(\\d+.\\d+.\\d+.\\d+\\))","State": "(\\[\\d\\])","DateTime": "(\\w+\\  \\d\\ \\d+:\\d+:\\d+)"}}`
	_ = rule2

	resp := internal.NewResponse()

	runtimeOps := &internal.RuntimeOptions{}

	runtimeOps.Unmarshal([]byte(rule1), json.Unmarshal)

	// historyCommandHeadPattern := `(\w+  \d \d+:\d+:\d+ \w+ \w+: \w+     \w+\/\d+        \d+-\d+-\d+ \d+:\d+ \(\d+.\d+.\d+.\d+\) \[\d+\]:)`
	// internal.VisitLogsAudit(
	// 	logParts,
	// 	resp,
	// 	[]byte(textSet[0]),
	// 	".*",
	// 	historyCommandHeadPattern,
	// 	`{"UserName":4,"DateTime":"(\w+\  \d\ \d+:\d+:\d+)","IpAddr":"(\(\d+.\d+.\d+.\d+\))","State":"(\[\d\])"}`,
	// 	"x86-64",
	// 	internal.SERVER,
	// )

	// //`Jan  7 01:20:40 bogon 1741: *Jan  7 01:21:44.906: %SYS-5-CONFIG_I: Configured from console by yicheng on vty8 (10.10.10.10)`
	// test2HeadPattern := `\w+  \d \d+:\d+:\d+ \w+ \d+\: \*\w+  \d \d+:\d+:\d+\.\d+: \%\w+-\d+-\w+:`
	// internal.VisitLogsAudit(
	// 	logParts,
	// 	resp,
	// 	[]byte(textSet[1]),
	// 	".*",
	// 	test2HeadPattern,
	// 	`{"UserName":4,"DateTime":"(\w+\  \d\ \d+:\d+:\d+)","IpAddr":"(\(\d+.\d+.\d+.\d+\))"}`,
	// 	"Cisco",
	// 	internal.SWITCH,
	// )

	internal.VisitLogsAudit2(logParts, []byte(textSet[0]), resp, runtimeOps)

	fmt.Fprintf(os.Stdout, "[DEBUG] %#v\n%#v\nresp:%#v", runtimeOps, runtimeOps.ColumnPattern, resp)
	//	time.Sleep(1 * time.Second)
}
