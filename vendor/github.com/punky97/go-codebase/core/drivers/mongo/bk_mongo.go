package mongo

import (
	"github.com/punky97/go-codebase/core/drivers/mongo/mgo_access_layout"
	"github.com/punky97/go-codebase/core/drivers/mongo/mgo_mock"
	core_models "github.com/punky97/go-codebase/exmsg/core/models"
	"context"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/spf13/cast"
)

// BkMongo represents mongo struct
type BkMongo struct {
	Session mgo_access_layout.SesstionAL
}

// Close mongo connection
func (c *BkMongo) Close() {
	c.Session.Close()
}

// GetDB -- get session
func (c *BkMongo) GetDB() mgo_access_layout.DatabaseAL {
	if session, ok := c.Session.(*mgo.Session); ok {
		return session.DB("")
	}
	if session, ok := c.Session.(*mgo_mock.Session); ok {
		return session.DB("")
	}
	return nil
}

// Deprecated: Please use NewSelect or NewUpdate instead
func (c *BkMongo) MGTool(collectionInfo CollectionInfo) *MGTool {
	c.Session.Refresh()
	return &MGTool{
		Collection:           &collectionInfo,
		Db:                   c.GetDB(),
		IncrementIdGenerator: c.incrementIDGenerator,
	}
}

// MGTool --
func (c *BkMongo) NewSelectCs(collectionInfo CollectionInfo, ctx context.Context, test bool) *MGTool {
	if !test {
		c.Session.Refresh()
	}
	mgt := &MGTool{
		Collection:           &collectionInfo,
		IncrementIdGenerator: c.incrementIDGenerator,
		ctx:                  ctx,
	}
	if !test {
		mgt.Db = c.GetDB()
	}
	mgt.initTool(false)
	return mgt
}

// MGTool --
func (c *BkMongo) NewSelect(collectionInfo CollectionInfo, ctx context.Context) *MGTool {
	return c.NewSelectCs(collectionInfo, ctx, false)
}

// MGTool --
func (c *BkMongo) NewUpdateCs(collectionInfo CollectionInfo, ctx context.Context, test bool) *MGTool {
	if !test {
		c.Session.Refresh()
	}
	mgt := &MGTool{
		Collection:           &collectionInfo,
		IncrementIdGenerator: c.incrementIDGenerator,
		ctx:                  ctx,
	}
	if !test {
		mgt.Db = c.GetDB()
	}
	mgt.initTool(true)
	return mgt
}

// MGTool --
func (c *BkMongo) NewUpdate(collectionInfo CollectionInfo, ctx context.Context) *MGTool {
	return c.NewUpdateCs(collectionInfo, ctx, false)
}

func (bkMongo *BkMongo) incrementIDGenerator(collection string) (newID int64, err error) {
	mgTool := bkMongo.MGTool(doctrineManager.Collection)
	result, err := mgTool.FindAndModify(
		bson.M{"_id": collection},
		mgo.Change{
			Update: bson.M{
				"$inc": bson.M{"current_id": 1},
			},
			ReturnNew: true,
		},
	)

	if v, ok := result["current_id"]; ok {
		newID = cast.ToInt64(v)
	}

	if err != nil && err.Error() == "not found" {
		_, err = mgTool.Create(&core_models.Doctrine{
			Id:        collection,
			CurrentId: 1,
		})

		newID = 1

		if err != nil {
			return
		}
	}

	return
}
