package model

import (
	"net/http"
	"time"

	"github.com/kekim-go/Author/constant"
	"github.com/kekim-go/Author/database"
	errors "github.com/kekim-go/Author/error"
	grpc_author "github.com/kekim-go/Protobuf/gen/proto/author"
	"xorm.io/xorm"
)

// App Api 서비스 관리 모델
type App struct {
	Id        uint       `xorm:"pk"`
	NameSpace string     `xorm:"unique"`
	IsDel     bool       `xorm:index default 0`
	Version   int        `xorm:"version"`
	CreatedAt time.Time  `xorm:"created"`
	UpdatedAt time.Time  `xorm:"updated"`
	DeletedAt *time.Time `xorm:"deleted index"`

	Operations []Operation `xorm:"- extends"`
	Traffics   []Traffic   `xorm:"- extends"`
}

func (App) TableName() string {
	return "app"
}

func (a *App) KeyName() string {
	return constant.KeyApp + a.NameSpace
}

func (a *App) FindApp(orm *xorm.Engine) error {
	found, err := orm.Get(a)

	if err != nil {
		return errors.NewWithPrefix(err, "database error")
	}

	if !found {
		return errors.NewWithCode(http.StatusNotFound, "app not found")
	}

	return nil
}

func (a *App) Delete(orm *xorm.Engine) error {
	sql := "UPDATE app SET deleted_at = ?, is_del = 1 WHERE id = ?"
	if _, err := orm.Exec(sql, time.Now(), a.Id); err != nil {
		return err
	}

	return nil
}

func (a *App) DelRedis(rdb *database.RedisDB) {
	rdb.Delete(a.KeyName())
}

func NewAppByGrpc(req *grpc_author.AppReq) *App {
	app := &App{}
	app.Id = uint(req.AppId)
	app.NameSpace = req.NameSpace

	if len(req.Operations) > 0 {
		for _, operation := range req.Operations {
			app.Operations = append(app.Operations, Operation{
				Id:       uint(operation.OperationId),
				AppId:    app.Id,
				EndPoint: operation.EndPoint,
			})
		}
	}

	if len(req.Traffics) > 0 {
		for _, traffic := range req.Traffics {
			app.Traffics = append(app.Traffics, Traffic{
				AppId: app.Id,
				Unit:  traffic.Unit,
				Val:   uint(traffic.Value),
				Seq:   uint(traffic.Seq),
			})
		}
	}

	return app
}
