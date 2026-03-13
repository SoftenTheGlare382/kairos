module kairos/services/worker

go 1.25.0

require (
	github.com/redis/go-redis/v9 v9.17.2
	gorm.io/driver/mysql v1.6.0
	gorm.io/gorm v1.31.1
	kairos v0.0.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace kairos => ../..
