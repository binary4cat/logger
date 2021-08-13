# 介绍

此日志组件是对 [zap](https://github.com/uber-go/zap) 组件的简单封装，增加了日志写入到文件，日志文件按照大小循环等可配置项，简化使用。

# 使用方式

```golang
package main

import (
 "fmt"
 "github.com/binary4cat/logger"
 "time"
)

func main() {
 opt := logger.Options{
  NotStdout:  false,          // 不要输出到标准输出，默认输出
  Level:      logger.DebugLevel, // 打印日志级别
  Filename:   "./log.log",    // 日志文件，为空则只输出到标准输出
  MaxSize:    10,             // 日志文件最多写入多大，单位MB
  MaxBackups: 10,             // 保存多少份日志文件
  MaxAge:     20,             // 日志文件保存多长时间，单位天
  Compress:   false,          // 备份的日志文件是否压缩
 }

 // 接收一个配置对象和多个hook
 logger.InitLogger(&opt, logInfoHook)

 logger.Debug("debug")
 logger.Infof("current time %s", time.Now().Format("20060102"))
 // 输出不带任何附加信息的日志
 logger.Pure("11", 22, "33")
}

// 可以定义多个hook，在打印日志的时候做自定义操作，例如异步推送到ES数据等
func logInfoHook(info logger.LogInfo) error {
 fmt.Printf("正在打印日志：%#v\n", info)
 return nil
}
```

## 关于`Pure`方法

有时候我们可能需要输出纯粹的内容，而不是被日志组件格式化的内容，因为调用日志组件的打印方法会在我们的日志内容中追加时间、打印日志的代码位置等信息。但是有些时候我们希望只输出我们的日志内容即可，不需要追加额外的信息。

例如将gorm的日志打印进我们的`logger`日志系统中，因为gorm的日志会带有时间和调用代码的位置信息，如果调用`logger`其他打印日志方法可能会因为追加信息太多而造成混乱，所以这种情况就可以调用`Pure`或者`Puref`方法去打印原始的日志内容：

```golang
func InitDb() {
 if db, err := gorm.Open(config.DatabaseConf.DbName, config.DatabaseConf.DbSource); err != nil {
  logger.Errorf("初始化数据库发生错误：%v", err)
 }
 if config.DatabaseConf.EnvModel {
  db.LogMode(true)
 } else {
  db.LogMode(false)
 }
 db.SetLogger(gormLogger{})
 db.DB().SetConnMaxLifetime(time.Second * time.Duration(config.DatabaseConf.ConnMaxLifetime))
 db.DB().SetMaxIdleConns(config.DatabaseConf.MaxIdleConns)
 db.DB().SetMaxOpenConns(config.DatabaseConf.MaxOpenConns)
}

type gormLogger struct{}

func (gormLogger) Print(values ...interface{}) {
 logger.Pure(gorm.LogFormatter(values...)...)
}
```
