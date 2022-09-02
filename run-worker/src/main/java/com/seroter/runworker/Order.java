package com.seroter.runworker;

public class Order {
    private Integer OrderId;
    public Integer getOrderId() {
        return OrderId;
    }
    public void setOrderId(Integer orderId) {
        OrderId = orderId;
    }
    private Integer CustomerId;
    public Integer getCustomerId() {
        return CustomerId;
    }
    public void setCustomerId(Integer customerId) {
        CustomerId = customerId;
    }
    private Integer ProductId;
    public Integer getProductId() {
        return ProductId;
    }
    public void setProductId(Integer productId) {
        ProductId = productId;
    }
    private Integer Quantity;
    public Integer getQuantity() {
        return Quantity;
    }
    public void setQuantity(Integer quantity) {
        Quantity = quantity;
    }
    private String FulfillmentHub;
    public String getFulfillmentHub() {
        return FulfillmentHub;
    }
    public void setFulfillmentHub(String fulfillmentHub) {
        FulfillmentHub = fulfillmentHub;
    }
    private String Status;
    public String getStatus() {
        return Status;
    }
    public void setStatus(String status) {
        Status = status;
    }
    private String OrderDate;
    public String getOrderDate() {
        return OrderDate;
    }
    public void setOrderDate(String orderDate) {
        OrderDate = orderDate;
    }
    
}
