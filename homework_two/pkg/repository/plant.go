package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"rentit/pkg/domain"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	redisKey = "app:plant"
)

type PlantRepository struct {
	mongoClient *mongo.Client
	db          *sql.DB
	redis       *redis.Client
}

func NewPlantRepository(mongoClient *mongo.Client, db *sql.DB, redis *redis.Client) *PlantRepository {
	return &PlantRepository{
		mongoClient: mongoClient,
		db:          db,
		redis:       redis,
	}
}

func (r *PlantRepository) GetAll() ([]*domain.Plant, error) {
	log.Printf("received get all request")

	// checking cache
	cached, cErr := r.redis.Exists(context.Background(), redisKey).Result()

	if cErr != nil {
		log.Println("Error checking plants in cache")
	}

	if cached == 1 {
		log.Println("Retrieving plants from cache")

		// results are not in the original order, should be ok as it is not specified in the task
		res, err := r.redis.HGetAll(context.Background(), redisKey).Result()

		if err != nil {
			log.Println("Failed to get plants from cache")
		}

		plants := []*domain.Plant{}
		for _, stringValue := range res {
			b := &domain.Plant{}
			err := json.Unmarshal([]byte(stringValue), b)
			if err != nil {
				log.Println("Error decoding plant from cache")
				break
			}
			plants = append(plants, b)
		}
		return plants, nil

	} else {
		log.Println("Plants not found in cache, querying DB")
	}

	// postgres
	query := "SELECT p.plant_id, pt.plant_type_name, p.plant_daily_rental_price, p.plant_name FROM plant p LEFT JOIN plant_type pt ON pt.plant_type_id = p.plant_type_id;"
	rows, err := r.db.QueryContext(context.Background(), query)

	//mongo
	mongoList, err := r.mongoClient.Database("Plants").Collection("plant").Find(context.Background(), bson.D{})
	results := []*domain.Plant{}
	if err != nil {

		fmt.Println(err)

	}
	for mongoList.Next(context.Background()) {

		elem := &domain.Plant{}
		err := mongoList.Decode(&elem)
		if err != nil {
			log.Fatal(err)
		}

		results = append(results, elem)
	}
	mongoList.Close(context.Background())

	// Processing all the queries
	if err != nil {
		return nil, fmt.Errorf("Error getting all plants from the DB, %v", err)
	}

	plants := make([]*domain.Plant, 0)
	for rows.Next() {
		p := &domain.Plant{}
		err := rows.Scan(&p.Plant_id, &p.Plant_type_name, &p.Plant_daily_rental_price, &p.Plant_name)
		if err != nil {
			return nil, fmt.Errorf("Error scaning query, %v", err)
		}
		plants = append(plants, p)

		// cache the plant
		_, cErr := r.redis.HSetNX(context.Background(), redisKey, string(p.Plant_id), p).Result()

		if cErr != nil {
			log.Println(cErr.Error())
			log.Println("Failed to cache a plant, idk what to do about it..")
		}

	}

	err = rows.Close()
	if err != nil {
		return nil, fmt.Errorf("could not close rows, %v", err)
	}

	// combine mongo and postgres
	for _, elem := range results {
		plants = append(plants, elem)
	}
	return plants, nil

}

func (r *PlantRepository) EstimateRental(queryStruct *domain.GetInfoQuery) (float32, error) {

	log.Printf("Received an estimation request")
	query := "select p.plant_daily_rental_price * EXTRACT(DAY FROM ($3::timestamp - $2::timestamp)) from plant p WHERE p.plant_name ILIKE '%' || $1 || '%';"

	row := r.db.QueryRowContext(context.Background(), query, &queryStruct.Plant_name, &queryStruct.Start_date, &queryStruct.End_date)

	if row == nil {
		return 0, fmt.Errorf("error estimating rental, %v", row)
	}

	var estimation float32

	err := row.Scan(&estimation)

	if err != nil {
		return 0, fmt.Errorf("error estimating rental, %v", err)
	}

	return estimation, nil

}

func (r *PlantRepository) AvailabilityCheck(queryStruct *domain.GetInfoQuery) (bool, error) {

	log.Printf("Received an availability request")

	query :=
		`
	select CASE
    when exists(
            SELECT 1
            FROM booking b
            WHERE b.plant_id = (SELECT p.plant_id FROM plant p WHERE p.plant_name ILIKE '%' || $1 || '%')
              AND (($2::timestamp >= b.start_date AND $3::timestamp <= b.end_date) OR
                   ($2::timestamp <= b.start_date AND $3::timestamp >= b.start_date) OR
                   ($2::timestamp <= b.end_date AND $3::timestamp >= b.end_date))
        )
    then false
    else true
	end`

	row := r.db.QueryRowContext(context.Background(), query, &queryStruct.Plant_name, &queryStruct.Start_date, &queryStruct.End_date)

	if row == nil {
		return false, fmt.Errorf("error checking for availability, %v", row)
	}

	var availability bool

	err := row.Scan(&availability)

	if err != nil {
		return false, fmt.Errorf("error checking for availability, %v", err)
	}

	return availability, nil

}
