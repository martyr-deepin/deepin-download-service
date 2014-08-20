# deepin-download-service

**How to build and install**


```
./build
```


**DBus接口文档**

**1 com.deepin.download.service**

**1.1 方法**

1.1.1. AddTask(taskName string, urls []string, sizes []int64, md5s []string, storeDir string) (taskid string)
    添加下载任务到下载队列中

    taskname:   任务名称
    urls:       下载的地址列表
    sizes:      下载文件大小列表（在服务器无法获取大小时生效，若能从服务器获得文件大小，则忽略该属性）
    md5s:       下载文件MD5列表（若不为空，则下载完成后会验证md5）
    storeDir:   存储位置
    taskid:     返回用于标示本次下载的任务id

1.1.2 PauseTask（taskid string）

    暂停任务

1.1.3 ResumeTask（taskid string）

    恢复暂停的任务

1.1.4 SoptTask（taskid string）

    停止/取消任务，停止任务后，任务会被删除

**1.2 信号**

1.2.1 Wait func(taskid string)

	任务下载未开始,处于等待准备状态时发出

	taskid: 任务id

1.2.2 Start func(taskid string)

	任务下载开始时发出

	taskid: 任务id

1.2.3 Update func(taskid string, progress int32, speed int32, finish int32, total int32, downloadSize int64, taotalSize int64)

    每秒钟针对每个任务发出一个更新信号

    taskid: 任务id
    process: 下载进度0~100
    speeds 下载速度 Bit/s
    finish 下载完成的url数目
    total  总共下载的url数目
    downloadSize 已经下载的数据 Byte
    totalSize 总共需要下载的数据 Byte

1.2.4 Finish func(taskid string)

	任务完成时发出

1.2.5 Stop func(taskid string)

	任务停止时发出, 任务Stop后会被立即删除，无法再获得任务信息，
	一般发出Stop信号，则任务任务失败

	taskid: 任务id

1.2.6 Pause func(taskid string)

	任务暂停时发出

	taskid: 任务id

1.2.7 Resume func(taskid string)

	任务继续时发出

	taskid: 任务id

1.2.8 Error func(taskid string, errcode int32, errstr string)

	发生错误时发出

	taskid: 任务id, 为空时表示与下载无关错误
	errcode: 错误码
	errStr: 错误描述

