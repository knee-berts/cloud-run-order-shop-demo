CREATE TABLE Orders (
    OrderId INT64 NOT NULL,    
    ProductId INT64,
    CustomerId INT64,
    Quantity INT64,
    OrderDate STRING(60),   
    FulfillmentHub STRING(3),
    LastUpdateTime TIMESTAMP NOT NULL OPTIONS(allow_commit_timestamp=true),
    Status STRING(20),
) PRIMARY KEY(OrderId);

CREATE TABLE OrdersHistory (
    OrderId INT64 NOT NULL,    
    ProductId INT64,
    CustomerId INT64,
    Quantity INT64,
    OrderDate STRING(60),   
    FulfillmentHub STRING(3),
    TimeStamp TIMESTAMP NOT NULL OPTIONS(allow_commit_timestamp=true),
    Status STRING(20),
) PRIMARY KEY(OrderId, TimeStamp),
INTERLEAVE IN PARENT Orders ON DELETE NO ACTION