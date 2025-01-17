package repositories

import (
	"context"
	"fmt"

	"github.com/mehdihadeli/go-ecommerce-microservices/internal/pkg/logger"
	"github.com/mehdihadeli/go-ecommerce-microservices/internal/pkg/mongodb"
	"github.com/mehdihadeli/go-ecommerce-microservices/internal/pkg/otel/tracing"
	"github.com/mehdihadeli/go-ecommerce-microservices/internal/pkg/otel/tracing/attribute"
	"github.com/mehdihadeli/go-ecommerce-microservices/internal/pkg/utils"
	"github.com/mehdihadeli/go-ecommerce-microservices/internal/services/orderservice/internal/orders/contracts/repositories"
	"github.com/mehdihadeli/go-ecommerce-microservices/internal/services/orderservice/internal/orders/models/orders/read_models"

	"emperror.dev/errors"
	uuid "github.com/satori/go.uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	attribute2 "go.opentelemetry.io/otel/attribute"
)

const (
	orderCollection = "orders"
)

type mongoOrderReadRepository struct {
	log          logger.Logger
	mongoOptions *mongodb.MongoDbOptions
	mongoClient  *mongo.Client
	tracer       tracing.AppTracer
}

func NewMongoOrderReadRepository(
	log logger.Logger,
	cfg *mongodb.MongoDbOptions,
	mongoClient *mongo.Client,
	tracer tracing.AppTracer,
) repositories.OrderMongoRepository {
	return &mongoOrderReadRepository{
		log:          log,
		mongoOptions: cfg,
		mongoClient:  mongoClient,
		tracer:       tracer,
	}
}

func (m mongoOrderReadRepository) GetAllOrders(
	ctx context.Context,
	listQuery *utils.ListQuery,
) (*utils.ListResult[*read_models.OrderReadModel], error) {
	ctx, span := m.tracer.Start(ctx, "mongoOrderReadRepository.GetAllOrders")
	defer span.End()

	collection := m.mongoClient.Database(m.mongoOptions.Database).Collection(orderCollection)

	result, err := mongodb.Paginate[*read_models.OrderReadModel](ctx, listQuery, collection, nil)
	if err != nil {
		return nil, tracing.TraceErrFromSpan(
			span,
			errors.WrapIf(
				err,
				"[mongoOrderReadRepository_GetAllOrders.Paginate] error in the paginate",
			),
		)
	}

	m.log.Infow(
		"[mongoOrderReadRepository.GetAllOrders] orders loaded",
		logger.Fields{"OrdersResult": result},
	)

	span.SetAttributes(attribute.Object("OrdersResult", result))

	return result, nil
}

func (m mongoOrderReadRepository) SearchOrders(
	ctx context.Context,
	searchText string,
	listQuery *utils.ListQuery,
) (*utils.ListResult[*read_models.OrderReadModel], error) {
	ctx, span := m.tracer.Start(ctx, "mongoOrderReadRepository.SearchOrders")
	span.SetAttributes(attribute2.String("SearchText", searchText))
	defer span.End()

	collection := m.mongoClient.Database(m.mongoOptions.Database).Collection(orderCollection)

	filter := bson.D{
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "name", Value: primitive.Regex{Pattern: searchText, Options: "gi"}}},
			bson.D{
				{Key: "description", Value: primitive.Regex{Pattern: searchText, Options: "gi"}},
			},
		}},
	}

	result, err := mongodb.Paginate[*read_models.OrderReadModel](ctx, listQuery, collection, filter)
	if err != nil {
		return nil, tracing.TraceErrFromSpan(
			span,
			errors.WrapIf(
				err,
				"[mongoOrderReadRepository_SearchOrders.Paginate] error in the paginate",
			),
		)
	}
	span.SetAttributes(attribute.Object("OrdersResult", result))

	m.log.Infow(
		fmt.Sprintf(
			"[mongoOrderReadRepository.SearchOrders] orders loaded for search term '%s'",
			searchText,
		),
		logger.Fields{"OrdersResult": result},
	)

	return result, nil
}

func (m mongoOrderReadRepository) GetOrderById(
	ctx context.Context,
	id uuid.UUID,
) (*read_models.OrderReadModel, error) {
	ctx, span := m.tracer.Start(ctx, "mongoOrderReadRepository.GetOrderById")
	span.SetAttributes(attribute2.String("Id", id.String()))
	defer span.End()

	collection := m.mongoClient.Database(m.mongoOptions.Database).Collection(orderCollection)

	var order read_models.OrderReadModel
	if err := collection.FindOne(ctx, bson.M{"_id": id.String()}).Decode(&order); err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, tracing.TraceErrFromSpan(
			span,
			errors.WrapIf(
				err,
				fmt.Sprintf(
					"[mongoOrderReadRepository_GetOrderById.FindOne] can't find the order with id %s into the database.",
					id,
				),
			),
		)
	}
	span.SetAttributes(attribute.Object("Order", order))

	m.log.Infow(
		fmt.Sprintf("[mongoOrderReadRepository.GetOrderById] order with id %s laoded", id.String()),
		logger.Fields{"Order": order, "Id": id},
	)

	return &order, nil
}

func (m mongoOrderReadRepository) GetOrderByOrderId(
	ctx context.Context,
	orderId uuid.UUID,
) (*read_models.OrderReadModel, error) {
	ctx, span := m.tracer.Start(ctx, "mongoOrderReadRepository.GetOrderByOrderId")
	span.SetAttributes(attribute2.String("OrderId", orderId.String()))
	defer span.End()

	collection := m.mongoClient.Database(m.mongoOptions.Database).Collection(orderCollection)

	var order read_models.OrderReadModel
	if err := collection.FindOne(ctx, bson.M{"orderId": orderId.String()}).Decode(&order); err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, tracing.TraceErrFromSpan(
			span,
			errors.WrapIf(
				err,
				fmt.Sprintf(
					"[mongoOrderReadRepository_GetOrderById.FindOne] can't find the order with orderId %s into the database.",
					orderId.String(),
				),
			),
		)
	}
	span.SetAttributes(attribute.Object("Order", order))

	m.log.Infow(
		fmt.Sprintf(
			"[mongoOrderReadRepository.GetOrderById] order with orderId %s laoded",
			orderId.String(),
		),
		logger.Fields{"Order": order, "orderId": orderId},
	)

	return &order, nil
}

func (m mongoOrderReadRepository) CreateOrder(
	ctx context.Context,
	order *read_models.OrderReadModel,
) (*read_models.OrderReadModel, error) {
	ctx, span := m.tracer.Start(ctx, "mongoOrderReadRepository.CreateOrder")
	defer span.End()

	collection := m.mongoClient.Database(m.mongoOptions.Database).Collection(orderCollection)
	_, err := collection.InsertOne(ctx, order, &options.InsertOneOptions{})
	if err != nil {
		return nil, tracing.TraceErrFromSpan(
			span,
			errors.WrapIf(
				err,
				"[mongoOrderReadRepository_CreateOrder.InsertOne] error in the inserting order into the database.",
			),
		)
	}
	span.SetAttributes(attribute.Object("Order", order))

	m.log.Infow(
		fmt.Sprintf(
			"[mongoOrderReadRepository.CreateOrder] order with id '%s' created",
			order.OrderId,
		),
		logger.Fields{"Order": order, "Id": order.OrderId},
	)

	return order, nil
}

func (m mongoOrderReadRepository) UpdateOrder(
	ctx context.Context,
	order *read_models.OrderReadModel,
) (*read_models.OrderReadModel, error) {
	ctx, span := m.tracer.Start(ctx, "mongoOrderReadRepository.UpdateOrder")
	defer span.End()

	collection := m.mongoClient.Database(m.mongoOptions.Database).Collection(orderCollection)

	ops := options.FindOneAndUpdate()
	ops.SetReturnDocument(options.After)
	ops.SetUpsert(true)

	var updated read_models.OrderReadModel
	if err := collection.FindOneAndUpdate(ctx, bson.M{"_id": order.OrderId}, bson.M{"$set": order}, ops).Decode(&updated); err != nil {
		return nil, tracing.TraceErrFromSpan(
			span,
			errors.WrapIf(
				err,
				fmt.Sprintf(
					"[mongoOrderReadRepository_UpdateOrder.FindOneAndUpdate] error in updating order with id %s into the database.",
					order.OrderId,
				),
			),
		)
	}
	span.SetAttributes(attribute.Object("Order", order))

	m.log.Infow(
		fmt.Sprintf(
			"[mongoOrderReadRepository.UpdateOrder] order with id '%s' updated",
			order.OrderId,
		),
		logger.Fields{"Order": order, "Id": order.OrderId},
	)

	return &updated, nil
}

func (m mongoOrderReadRepository) DeleteOrderByID(ctx context.Context, uuid uuid.UUID) error {
	ctx, span := m.tracer.Start(ctx, "mongoOrderReadRepository.DeleteOrderByID")
	span.SetAttributes(attribute2.String("Id", uuid.String()))
	defer span.End()

	collection := m.mongoClient.Database(m.mongoOptions.Database).Collection(orderCollection)

	if err := collection.FindOneAndDelete(ctx, bson.M{"_id": uuid.String()}).Err(); err != nil {
		return tracing.TraceErrFromSpan(span, errors.WrapIf(err, fmt.Sprintf(
			"[mongoOrderReadRepository_DeleteOrderByID.FindOneAndDelete] error in deleting order with id %d from the database.",
			uuid,
		)))
	}

	m.log.Infow(
		fmt.Sprintf("[mongoOrderReadRepository.DeleteOrderByID] order with id %s deleted", uuid),
		logger.Fields{"Id": uuid},
	)

	return nil
}
