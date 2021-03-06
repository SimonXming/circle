package datastore

import (
	"github.com/simonshyu/circle/model"
	"github.com/simonshyu/circle/store/datastore/sql"
	"github.com/russross/meddler"
)

func (db *datastore) TaskList() ([]*model.Task, error) {
	stmt := sql.Lookup(db.driver, "task-list")
	data := []*model.Task{}
	err := meddler.QueryAll(db, &data, stmt)
	return data, err
}

func (db *datastore) TaskInsert(task *model.Task) error {
	return meddler.Insert(db, taskTable, task)
}

func (db *datastore) TaskDelete(id string) error {
	stmt := sql.Lookup(db.driver, "task-delete")
	_, err := db.Exec(stmt, id)
	return err
}

const taskTable = "tasks"
