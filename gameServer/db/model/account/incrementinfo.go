package account

import (
	"fmt"
	"mj/gameServer/db"

	"github.com/jmoiron/sqlx"
	"github.com/lovelly/leaf/log"
)

//This file is generate by scripts,don't edit it

//incrementinfo
//

// +gen *
type Incrementinfo struct {
	IncrementName  string `db:"increment_name" json:"increment_name"`   //
	IncrementValue int64  `db:"increment_value" json:"increment_value"` //
}

type incrementinfoOp struct{}

var IncrementinfoOp = &incrementinfoOp{}
var DefaultIncrementinfo = &Incrementinfo{}

// 按主键查询. 注:未找到记录的话将触发sql.ErrNoRows错误，返回nil, false
func (op *incrementinfoOp) Get(increment_name string) (*Incrementinfo, bool) {
	obj := &Incrementinfo{}
	sql := "select * from incrementinfo where increment_name=? "
	err := db.AccountDB.Get(obj, sql,
		increment_name,
	)

	if err != nil {
		log.Error("Get data error:%v", err.Error())
		return nil, false
	}
	return obj, true
}
func (op *incrementinfoOp) SelectAll() ([]*Incrementinfo, error) {
	objList := []*Incrementinfo{}
	sql := "select * from incrementinfo "
	err := db.AccountDB.Select(&objList, sql)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	return objList, nil
}

func (op *incrementinfoOp) QueryByMap(m map[string]interface{}) ([]*Incrementinfo, error) {
	result := []*Incrementinfo{}
	var params []interface{}

	sql := "select * from incrementinfo where 1=1 "
	for k, v := range m {
		sql += fmt.Sprintf(" and %s=? ", k)
		params = append(params, v)
	}
	err := db.AccountDB.Select(&result, sql, params...)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	return result, nil
}

func (op *incrementinfoOp) GetByMap(m map[string]interface{}) (*Incrementinfo, error) {
	lst, err := op.QueryByMap(m)
	if err != nil {
		return nil, err
	}
	if len(lst) > 0 {
		return lst[0], nil
	}
	return nil, nil
}

/*
func (i *Incrementinfo) Insert() error {
    err := db.AccountDBMap.Insert(i)
    if err != nil{
		log.Error("Insert sql error:%v, data:%v", err.Error(),i)
        return err
    }
}
*/

// 插入数据，自增长字段将被忽略
func (op *incrementinfoOp) Insert(m *Incrementinfo) (int64, error) {
	return op.InsertTx(db.AccountDB, m)
}

// 插入数据，自增长字段将被忽略
func (op *incrementinfoOp) InsertTx(ext sqlx.Ext, m *Incrementinfo) (int64, error) {
	sql := "insert into incrementinfo(increment_name,increment_value) values(?,?)"
	result, err := ext.Exec(sql,
		m.IncrementName,
		m.IncrementValue,
	)
	if err != nil {
		log.Error("InsertTx sql error:%v, data:%v", err.Error(), m)
		return -1, err
	}
	affected, _ := result.LastInsertId()
	return affected, nil
}

//存在就更新， 不存在就插入
func (op *incrementinfoOp) InsertUpdate(obj *Incrementinfo, m map[string]interface{}) error {
	sql := "insert into incrementinfo(increment_name,increment_value) values(?,?) ON DUPLICATE KEY UPDATE "
	var params = []interface{}{obj.IncrementName,
		obj.IncrementValue,
	}
	var set_sql string
	for k, v := range m {
		if set_sql != "" {
			set_sql += ","
		}
		set_sql += fmt.Sprintf(" %s=? ", k)
		params = append(params, v)
	}

	_, err := db.AccountDB.Exec(sql+set_sql, params...)
	return err
}

/*
func (i *Incrementinfo) Update()  error {
    _,err := db.AccountDBMap.Update(i)
    if err != nil{
		log.Error("update sql error:%v, data:%v", err.Error(),i)
        return err
    }
}
*/

// 用主键(属性)做条件，更新除主键外的所有字段
func (op *incrementinfoOp) Update(m *Incrementinfo) error {
	return op.UpdateTx(db.AccountDB, m)
}

// 用主键(属性)做条件，更新除主键外的所有字段
func (op *incrementinfoOp) UpdateTx(ext sqlx.Ext, m *Incrementinfo) error {
	sql := `update incrementinfo set increment_value=? where increment_name=?`
	_, err := ext.Exec(sql,
		m.IncrementValue,
		m.IncrementName,
	)

	if err != nil {
		log.Error("update sql error:%v, data:%v", err.Error(), m)
		return err
	}

	return nil
}

// 用主键做条件，更新map里包含的字段名
func (op *incrementinfoOp) UpdateWithMap(increment_name string, m map[string]interface{}) error {
	return op.UpdateWithMapTx(db.AccountDB, increment_name, m)
}

// 用主键做条件，更新map里包含的字段名
func (op *incrementinfoOp) UpdateWithMapTx(ext sqlx.Ext, increment_name string, m map[string]interface{}) error {

	sql := `update incrementinfo set %s where 1=1 and increment_name=? ;`

	var params []interface{}
	var set_sql string
	for k, v := range m {
		if set_sql != "" {
			set_sql += ","
		}
		set_sql += fmt.Sprintf(" %s=? ", k)
		params = append(params, v)
	}
	params = append(params, increment_name)
	_, err := ext.Exec(fmt.Sprintf(sql, set_sql), params...)
	return err
}

/*
func (i *Incrementinfo) Delete() error{
    _,err := db.AccountDBMap.Delete(i)
	log.Error("Delete sql error:%v", err.Error())
    return err
}
*/
// 根据主键删除相关记录
func (op *incrementinfoOp) Delete(increment_name string) error {
	return op.DeleteTx(db.AccountDB, increment_name)
}

// 根据主键删除相关记录,Tx
func (op *incrementinfoOp) DeleteTx(ext sqlx.Ext, increment_name string) error {
	sql := `delete from incrementinfo where 1=1
        and increment_name=?
        `
	_, err := ext.Exec(sql,
		increment_name,
	)
	return err
}

// 返回符合查询条件的记录数
func (op *incrementinfoOp) CountByMap(m map[string]interface{}) (int64, error) {

	var params []interface{}
	sql := `select count(*) from incrementinfo where 1=1 `
	for k, v := range m {
		sql += fmt.Sprintf(" and  %s=? ", k)
		params = append(params, v)
	}
	count := int64(-1)
	err := db.AccountDB.Get(&count, sql, params...)
	if err != nil {
		log.Error("CountByMap  error:%v data :%v", err.Error(), m)
		return 0, err
	}
	return count, nil
}

func (op *incrementinfoOp) DeleteByMap(m map[string]interface{}) (int64, error) {
	return op.DeleteByMapTx(db.AccountDB, m)
}

func (op *incrementinfoOp) DeleteByMapTx(ext sqlx.Ext, m map[string]interface{}) (int64, error) {
	var params []interface{}
	sql := "delete from incrementinfo where 1=1 "
	for k, v := range m {
		sql += fmt.Sprintf(" and %s=? ", k)
		params = append(params, v)
	}
	result, err := ext.Exec(sql, params...)
	if err != nil {
		return -1, err
	}
	return result.RowsAffected()
}
