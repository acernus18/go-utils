package business

import (
	"crypto/md5"
	"database/sql"
	"encoding/csv"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Info struct {
	Id            int
	OrderId       string
	SoldTime      mysql.NullTime
	GemID         string
	GamName       string
	GemType       string
	GemNumber     int
	GemWeight     float32
	GemUnitPrice  float32
	GemCost       float32
	GemTotalPrice float32
	GemRealPrice  float32
	SoldTo        string
	Other         string
}

const (
	CodePattern = `^.*?\W?([VHKLMNRSTUBCDEF]{2,})\W?.*$`
	CodeSet     = "VHKLMNRSTU"
	CodeSuffix  = "BCDEF"
	DataSource  = "root:dev#pass@tcp(120.78.175.39:30001)/db_jewelry"
)

type SalesRecord struct {
	ID        int       `db:"id"`
	Digest    string    `db:"digest"`
	SoldID    string    `db:"sold_id"`
	SoldTime  time.Time `db:"sold_time"`
	SoldTo    string    `db:"sold_to"`
	GemID     string    `db:"gem_id"`
	GemInfo   string    `db:"gem_info"`
	GemType   int       `db:"gem_type"`
	Number    int       `db:"number"`
	Weight    float32   `db:"weight"`
	UnitCost  float32   `db:"unit_cost"`
	UnitPrice float32   `db:"unit_price"`
	RealPrice float32   `db:"real_price"`
}

func (record *SalesRecord) generateDigest() {
	t := record.SoldID
	c := record.SoldTo
	i := record.GemID
	w := strconv.FormatFloat(float64(record.Weight), 'f', -1, 32)
	p := strconv.FormatFloat(float64(record.UnitPrice), 'f', -1, 32)
	information := []byte(fmt.Sprintf("%s%s%s%s%s", t, c, i, w, p))
	record.Digest = fmt.Sprintf("%x", md5.Sum(information))
}

func (record *SalesRecord) parseCost() {
	codePattern := regexp.MustCompile(CodePattern)
	if !codePattern.MatchString(record.GemInfo) {
		record.UnitCost = -1
		return
	}
	code := codePattern.FindStringSubmatch(record.GemInfo)[1]
	if code == "MM" {
		record.UnitCost = -1
		return
	}

	unitCost := 0
	for i := range code {
		char := string(code[i])
		if !strings.ContainsAny(char, CodeSuffix) {
			unitCost = unitCost*10 + strings.Index(CodeSet, string(code[i]))
		} else {
			unitCost *= 10 * (strings.Index(CodeSuffix, char) + 2 - len(string(unitCost)))
			break
		}
	}

	if unitCost < 10 {
		record.UnitCost = -1
	} else {
		record.UnitCost = float32(unitCost)
	}
}

func (record *SalesRecord) InsertStatement() (string, error) {
	soldTime := record.SoldTime.Format("2006-01-02")
	builder := new(strings.Builder)
	if _, e := builder.WriteString("insert into tb_sales_record values (NULL, "); e != nil {
		return "", e
	}
	if _, e := builder.WriteString(fmt.Sprintf("'%s', ", record.Digest)); e != nil {
		return "", e
	}
	if _, e := builder.WriteString(fmt.Sprintf("'%s', ", record.SoldID)); e != nil {
		return "", e
	}
	if _, e := builder.WriteString(fmt.Sprintf("'%s', ", soldTime)); e != nil {
		return "", e
	}
	if _, e := builder.WriteString(fmt.Sprintf("'%s', ", record.SoldTo)); e != nil {
		return "", e
	}
	if _, e := builder.WriteString(fmt.Sprintf("'%s', ", record.GemID)); e != nil {
		return "", e
	}
	if _, e := builder.WriteString(fmt.Sprintf("'%s', ", record.GemInfo)); e != nil {
		return "", e
	}
	if _, e := builder.WriteString(fmt.Sprintf("%d, ", record.GemType)); e != nil {
		return "", e
	}
	if _, e := builder.WriteString(fmt.Sprintf("%d, ", record.Number)); e != nil {
		return "", e
	}
	if _, e := builder.WriteString(fmt.Sprintf("%.3f, ", record.Weight)); e != nil {
		return "", e
	}
	if _, e := builder.WriteString(fmt.Sprintf("%.3f, ", record.UnitCost)); e != nil {
		return "", e
	}
	if _, e := builder.WriteString(fmt.Sprintf("%.3f, ", record.UnitPrice)); e != nil {
		return "", e
	}
	if _, e := builder.WriteString(fmt.Sprintf("%.3f", record.RealPrice)); e != nil {
		return "", e
	}
	if _, e := builder.WriteString(")"); e != nil {
		return "", e
	}
	return builder.String(), nil
}

func recogniseGemType(gemType string) int {
	mapper := map[string]int{
		"A红宝石":       0,
		"B斯里兰卡蓝宝石":   1,
		"C成品镶嵌":      2,
		"C蓝宝石":       3,
		"D代销裸石":      4,
		"d无烧鸽血红":     5,
		"EE有烧皇家蓝":    6,
		"E无烧皇家蓝":     7,
		"FF有烧矢车菊":    8,
		"F无烧矢车菊":     9,
		"G无烧蓝宝石":     10,
		"H无烧粉红蓝宝石":   11,
		"II无烧紫色蓝宝石":  12,
		"I无烧黄色蓝宝石":   13,
		"J代镶嵌加工费":    14,
		"J无烧帕帕拉恰蓝宝石": 15,
		"J镶嵌加工费":     16,
		"k无烧红宝石":     17,
		"K金素金":       18,
		"l有烧鸽血红":     19,
		"M祖母绿":       20,
		"N无油祖母绿":     21,
		"O哥伦比亚祖母绿":   22,
		"P帕帕拉恰蓝宝石":   23,
		"QQ紫色蓝宝石":    24,
		"Q粉红蓝宝石":     25,
		"R黄色蓝宝石":     26,
		"SS沙弗莱":      27,
		"T山东蓝宝石":     28,
		"U泰国蓝宝石":     29,
		"V猫眼":        30,
		"W变石猫眼":      31,
		"XX金绿宝石":     32,
		"X亚历山大变石":    33,
		"Y星光红宝石":     34,
		"ZZ珍珠":       35,
		"Z无烧橙色蓝宝石":   36,
		"Z星光蓝宝石":     37,
		"尖晶石":        38,
		"无烧变色蓝宝石":    39,
	}
	result, exist := mapper[gemType]
	if exist {
		return result
	} else {
		return -1
	}
}

func QueryOriginSoldList() ([]SalesRecord, error) {
	db, e := sql.Open("mysql", DataSource)
	if e != nil {
		return nil, e
	}

	rows, e := db.Query(`
		select 
		       id, order_id, sold_time, gem_id, gem_name, 
		       gem_type, gem_number, gem_weight, gem_unit_price, 
		       gem_total_price, gem_real_price, sold_to 
		from db_jewelry.tb_gem_sold_list`)
	if e != nil {
		return nil, e
	}

	records := make([]SalesRecord, 0)
	for rows.Next() {
		i := Info{}
		e := rows.Scan(&i.Id, &i.OrderId, &i.SoldTime, &i.GemID,
			&i.GamName, &i.GemType, &i.GemNumber, &i.GemWeight,
			&i.GemUnitPrice, &i.GemTotalPrice, &i.GemRealPrice, &i.SoldTo)
		if e != nil {
			return nil, e
		}
		salesRecord := SalesRecord{
			Digest:    "",
			SoldID:    i.OrderId,
			SoldTime:  i.SoldTime.Time,
			SoldTo:    i.SoldTo,
			GemID:     i.GemID,
			GemInfo:   i.GamName,
			GemType:   recogniseGemType(i.GemType),
			Number:    i.GemNumber,
			Weight:    i.GemWeight,
			UnitCost:  0,
			UnitPrice: i.GemUnitPrice,
			RealPrice: i.GemRealPrice,
		}
		salesRecord.generateDigest()
		salesRecord.parseCost()
		records = append(records, salesRecord)
	}

	if e := db.Close(); e != nil {
		return nil, e
	}
	return records, nil
}

func ReadSalesRecordFromCSV(filename string) ([]SalesRecord, error) {
	csvFile, e := os.Open(filename)
	if e != nil {
		return nil, e
	}

	salesRecords := make([]SalesRecord, 0)
	reader := csv.NewReader(csvFile)
	for {
		records, e := reader.Read()
		if e != nil && e == io.EOF {
			break
		} else if e != nil {
			return nil, e
		}

		soldTime, e := time.Parse("2006-01-02", records[0])
		if e != nil {
			return nil, e
		}

		gemType, e := strconv.ParseInt(records[5], 10, 32)
		if e != nil {
			return nil, e
		}

		number, e := strconv.ParseInt(records[6], 10, 32)
		if e != nil {
			return nil, e
		}

		weight, e := strconv.ParseFloat(records[7], 32)
		if e != nil {
			return nil, e
		}

		unitPrice, e := strconv.ParseFloat(records[8], 32)
		if e != nil {
			return nil, e
		}

		realPrice, e := strconv.ParseFloat(records[9], 32)
		if e != nil {
			return nil, e
		}

		temp := SalesRecord{
			ID:        0,
			Digest:    "",
			SoldID:    records[1],
			SoldTime:  soldTime,
			SoldTo:    records[2],
			GemID:     records[3],
			GemInfo:   records[4],
			GemType:   int(gemType),
			Number:    int(number),
			Weight:    float32(weight),
			UnitCost:  0.0,
			UnitPrice: float32(unitPrice),
			RealPrice: float32(realPrice),
		}
		temp.generateDigest()
		temp.parseCost()
		salesRecords = append(salesRecords, temp)
	}

	if e := csvFile.Close(); e != nil {
		return nil, e
	}

	return salesRecords, nil
}

func checkIfSalesRecordExist(db *sql.DB, records []SalesRecord) ([]SalesRecord, error) {
	output := make([]SalesRecord, 0)
	statement := "select count(0) from tb_sales_record where digest = ?"
	for i := range records {
		result := -1
		row := db.QueryRow(statement, records[i].Digest)
		if e := row.Scan(&result); e != nil {
			return nil, e
		}
		if result == 0 {
			output = append(output, records[i])
		}
	}
	return output, nil
}

func SaveSalesRecord(records []SalesRecord) error {
	// Distinct
	set := make(map[string]SalesRecord)
	for i := range records {
		set[records[i].Digest] = records[i]
	}

	distinctRecords := make([]SalesRecord, 0)
	for i := range set {
		distinctRecords = append(distinctRecords, set[i])
	}

	db, e := sql.Open("mysql", DataSource)
	if e != nil {
		return e
	}
	distinctRecords, e = checkIfSalesRecordExist(db, distinctRecords)
	if e != nil {
		return e
	}
	// Order by sold time
	sort.Slice(distinctRecords, func(i, j int) bool {
		return distinctRecords[i].SoldTime.Before(distinctRecords[j].SoldTime)
	})

	transaction, e := db.Begin()
	if e != nil {
		return e
	}
	for i := range distinctRecords {
		statement, e := distinctRecords[i].InsertStatement()
		fmt.Println(statement)
		if e != nil {
			if err := transaction.Rollback(); err != nil {
				return err
			} else {
				return e
			}
		}
		result, e := transaction.Exec(statement)
		if e != nil {
			if err := transaction.Rollback(); err != nil {
				return err
			} else {
				return e
			}
		}
		if result != nil {
			lastInsertID, e := result.LastInsertId()
			if e != nil {
				if err := transaction.Rollback(); err != nil {
					return err
				} else {
					return e
				}
			}
			rowsAffected, e := result.RowsAffected()
			if e != nil {
				if err := transaction.Rollback(); err != nil {
					return err
				} else {
					return e
				}
			}
			fmt.Printf("LastInsertID = %d, AffectedRow = %d\n", lastInsertID, rowsAffected)
		}
	}
	if e := transaction.Commit(); e != nil {
		if err := transaction.Rollback(); err != nil {
			return err
		} else {
			return e
		}
	}
	if e := db.Close(); e != nil {
		return e
	}
	return nil
}
