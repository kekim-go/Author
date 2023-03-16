package model

import (
	"fmt"
	"time"

	"github.com/kekim-go/Author/constant"
	"github.com/kekim-go/Author/database"
	errors "github.com/kekim-go/Author/error"
	"xorm.io/xorm"
)

type Traffic struct {
	Id        uint   `xorm:"pk autoincr"`
	AppId     uint   `xorm:"index index(with_seq)"`
	Unit      string `xorm:"varchar(10) index default 'd'"` //트래픽 단위(min:분, hour:시간, day:1일, month:1달)
	Val       uint
	Seq       uint      `xorm:"index(with_seq)"`
	CreatedAt time.Time `xorm:"created"`
	UpdatedAt time.Time `xorm:"updated"`
	DeletedAt *time.Time

	App App `xorm:"- extends"`
}

func (t *Traffic) KeyName() string {
	return fmt.Sprintf("%s%d:%s", constant.KeyAppTrafficPrefix, t.AppId, t.Unit)
}

func FindTrafficsByApp(orm *xorm.Engine, appId uint) ([]Traffic, error) {
	traffics := []Traffic{}

	err := orm.Where("app_id = ?", appId).Find(&traffics)

	if err != nil {
		return nil, errors.New("database error; " + err.Error())
	}

	return traffics, nil
}

func (t *Traffic) Delete(orm *xorm.Engine) error {
	if _, err := orm.ID(t.Id).Delete(t); err != nil {
		return err
	}

	return nil
}

func (t *Traffic) DelRedis(rdb *database.RedisDB) {
	rdb.Delete(t.KeyName())
}
