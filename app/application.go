package app

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/kekim-go/Author/app/ctx"
	"github.com/kekim-go/Author/constant"
	"github.com/kekim-go/Author/database"
	server "github.com/kekim-go/Author/grpc"
	"github.com/kekim-go/Author/model"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"xorm.io/xorm"
	"xorm.io/xorm/log"

	_ "github.com/jinzhu/gorm/dialects/mysql"
)

// Application define a mode of running app
type Application struct {
	Ctx     *ctx.Context
	Context context.Context
	server  *server.Server
}

// New constructor
func New(context context.Context) (*Application, error) {
	var err error

	a := new(Application)
	a.Ctx = new(ctx.Context)
	a.Context = context

	env := os.Getenv("AUTHOR_ENV")
	if len(env) > 0 && env == constant.ServiceProd {
		a.Ctx.Mode = constant.ServiceProd
	} else if len(env) > 0 && env == constant.ServiceStage {
		a.Ctx.Mode = constant.ServiceStage
	} else {
		a.Ctx.Mode = constant.ServiceDev
	}

	a.Ctx.DBConfigFileName = fmt.Sprintf("config/%s/database.yaml", a.Ctx.Mode)
	a.Ctx.RedisConfigFileName = fmt.Sprintf("config/%s/redis.yaml", a.Ctx.Mode)

	if err = a.initConfig(); err != nil {
		return nil, err
	}

	if err = a.initLogger(); err != nil {
		return nil, err
	}

	a.Ctx.Logger.Debug(fmt.Sprintf("Run author service in '%s' mode", a.Ctx.Mode))

	if err = a.initDB(); err != nil {
		return nil, err
	}

	a.initRedis(a.Context)

	return a, nil
}

// Run starts application
func (a *Application) Run(network, addr string) {
	a.server = server.New(a.Ctx, a.Context)
	if err := a.server.Run(network, addr); err != nil {
		a.Ctx.Logger.Info("Service Run failed")
		a.Ctx.Logger.Info(err.Error())
	}
}

func (a *Application) initConfig() error {
	var file []byte
	var err error

	if file, err = ioutil.ReadFile("config/config.yaml"); err != nil {
		return err
	}
	if err = yaml.Unmarshal(file, &a.Ctx.Config); err != nil {
		return err
	}

	// Load DB Config
	if file, err = ioutil.ReadFile(a.Ctx.DBConfigFileName); err != nil {
		return err
	}
	if err = yaml.Unmarshal(file, &a.Ctx.DBConfig); err != nil {
		return err
	}

	// Load Redis Config
	if file, err = ioutil.ReadFile(a.Ctx.RedisConfigFileName); err != nil {
		return err
	}
	if err = yaml.Unmarshal(file, &a.Ctx.RedisConfig); err != nil {
		return err
	}

	return nil
}

func (a *Application) initDB() error {
	var err error

	dbConfig := a.Ctx.DBConfig
	a.Ctx.Logger.Info("DB Config ==========================================")
	a.Ctx.Logger.Info(dbConfig)
	connectURL := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True",
		dbConfig.User, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.DBName)
	a.Ctx.Logger.Info(connectURL)

	if a.Ctx.Orm, err = xorm.NewEngine(dbConfig.DBType, connectURL); err != nil {
		return err
	}

	if a.Ctx.Mode == constant.ServiceDev {
		a.Ctx.Orm.ShowSQL(true)
		a.Ctx.Orm.Logger().SetLevel(log.LOG_DEBUG)
	}
	a.Ctx.Orm.SetMaxIdleConns(dbConfig.IdleConns)
	a.Ctx.Orm.SetMaxOpenConns(dbConfig.MaxOpenConns)

	//migrate
	err = a.migrateDB()

	return err
}

func (a *Application) migrateDB() error {
	var err error
	if err = a.Ctx.Orm.Sync2(new(model.App)); err != nil {
		return err
	}
	if err = a.Ctx.Orm.Sync2(new(model.Token)); err != nil {
		return err
	}
	if err = a.Ctx.Orm.Sync2(new(model.AppToken)); err != nil {
		return err
	}
	if err = a.Ctx.Orm.Sync2(new(model.AppTokenHistory)); err != nil {
		return err
	}
	if err = a.Ctx.Orm.Sync2(new(model.Operation)); err != nil {
		return err
	}
	if err = a.Ctx.Orm.Sync2(new(model.Traffic)); err != nil {
		return err
	}
	if err = a.Ctx.Orm.Sync2(new(model.Group)); err != nil {
		return err
	}
	if err = a.Ctx.Orm.Sync2(new(model.User)); err != nil {
		return err
	}
	if err = a.Ctx.Orm.Sync2(new(model.Role)); err != nil {
		return err
	}
	if err = a.Ctx.Orm.Sync2(new(model.UserRole)); err != nil {
		return err
	}

	return nil
}

func (a *Application) initRedis(context context.Context) {
	redisConfig := a.Ctx.RedisConfig
	redisClient := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", redisConfig.Addr, redisConfig.Port),
		Password:     redisConfig.Password,
		DB:           redisConfig.DB,
		MinIdleConns: redisConfig.MinIdleConns,
		PoolSize:     redisConfig.PoolSize,
	})

	a.Ctx.RedisDB = database.NewRedisDB(context, redisClient)
}

func (a *Application) initLogger() error {
	logger := logrus.New()

	if _, err := os.Stat("log"); os.IsNotExist(err) {
		os.Mkdir("log", 0777)
	}

	if a.Ctx.Mode == constant.ServiceDev {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	// logger.SetFormatter(&logrus.JSONFormatter{})
	logger.Out = os.Stdout
	//file, _ := os.OpenFile(a.Ctx.Config.LoggerConfig.FileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)
	//logger.Out = file

	a.Ctx.Logger = logger.WithFields(logrus.Fields{
		"tag": a.Ctx.Config.LoggerConfig.Tag,
		"id":  os.Getpid(),
	})

	return nil
}
