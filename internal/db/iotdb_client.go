package db

import (
	"OMEGA3-IOT/internal/config"
	"fmt"
	"github.com/apache/iotdb-client-go/client"
	"github.com/apache/iotdb-client-go/common"
	"log"
	"strings"
)

type IOTDBClient struct {
	//session      *client.Session
	SessionPool  client.SessionPool
	StorageGroup string
	//TODO 尚且需要确定SG是否唯一？
	Config config.Config
}

func (i *IOTDBClient) Close() {
	i.SessionPool.Close()
}

func (i *IOTDBClient) InsertRecord(deviceId string, measurements []string, dataTypes []client.TSDataType, values []interface{}, timestamp int64) (r *common.TSStatus, err error) {
	session, _ := i.SessionPool.GetSession()
	return session.InsertRecord(deviceId, measurements, dataTypes, values, timestamp)
}

func (i *IOTDBClient) ExecuteQuery(sql string, result *interface{}) error {
	session, err := i.SessionPool.GetSession()
	if err != nil {
		return fmt.Errorf("failed to get session from pool: %w", err)
	}
	defer i.SessionPool.PutBack(session)
	queryDataSet, err := session.ExecuteQueryStatement(sql, &i.Config.IoTDB.QueryTimeoutMs)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer queryDataSet.Close()
	*result = queryDataSet
	return nil
}
func (i *IOTDBClient) ExecuteNonQuery(sql string) (*common.TSStatus, error) {
	session, err := i.SessionPool.GetSession()
	if err != nil {
		return nil, fmt.Errorf("failed to get session from pool: %w", err)
	}
	defer i.SessionPool.PutBack(session)

	return session.ExecuteNonQueryStatement(sql)
}

// InsertRecordTyped 调用 session.InsertRecord，需要提供 dataTypes
func (i *IOTDBClient) InsertRecordTyped(deviceId string, measurements []string, dataTypes []client.TSDataType, values []interface{}, timestamp int64) error {
	session, err := i.SessionPool.GetSession()
	if err != nil {
		return fmt.Errorf("failed to get session from pool: %w", err)
	}
	defer i.SessionPool.PutBack(session)

	status, err := session.InsertRecord(deviceId, measurements, dataTypes, values, timestamp)
	return i.CheckError(status, err)
}

func (i *IOTDBClient) InitializeSchema() error {
	session, err := i.SessionPool.GetSession()
	if err != nil {
		return fmt.Errorf("failed to get session from pool: %w", err)
	}
	defer i.SessionPool.PutBack(session)

	storageGroup := "root.mm1"
	//latestStorageGroup := "root.mm1_latest"
	//实际上不需要latest，latest在MySQL里
	//TODO SG要拓展性
	i.setStorageGroup(storageGroup)
	//i.setStorageGroup(latestStorageGroup)
	return nil
}

// checkError: 返回nil则是成功
func (i *IOTDBClient) CheckError(status *common.TSStatus, err error) error {
	if err != nil {
		return fmt.Errorf("[IOTDB CLIENT] Error: failed to get session from pool: %w", err)
	}
	if status != nil {
		if status.GetCode() == client.MultipleError ||
			status.GetMessage() != "" && strings.Contains(status.GetMessage(), "already exists") {
			log.Printf("[IOTDB CLIENT] Info: %s ", status.GetMessage())
			return nil
		}
		if verifyErr := client.VerifySuccess(status); verifyErr != nil {
			return fmt.Errorf("[IOTDB CLIENT] Error: IoTDB operation failed: %w", verifyErr)
		}
	}
	return nil
}

func (i *IOTDBClient) setStorageGroup(storageGroup string) {
	session, err := i.SessionPool.GetSession()
	if err != nil {
		defer i.SessionPool.PutBack(session)
		log.Fatal("failed to get session from pool: %w", err)

	}

	i.StorageGroup = storageGroup
	status, err := session.SetStorageGroup(storageGroup)
	if checkErr := i.CheckError(status, err); checkErr != nil {
		log.Printf("[IOTDB CLIENT] Info: %s ", checkErr)
	}
}
