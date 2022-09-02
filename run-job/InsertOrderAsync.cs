using System;
using System.Threading.Tasks;
using Google.Cloud.Spanner.Data;

public class InsertOrderAsync
{
    public async Task InsertDataAsync()
    {
        string connectionString = $"Data Source={Environment.GetEnvironmentVariable("SPANNER_URI")}";
        
        string selectSql = "SELECT OrderId FROM Orders WHERE Status = 'SUBMITTED'";

        using (var connection = new SpannerConnection(connectionString))
        {
            await connection.OpenAsync();

            var selectCommand = connection.CreateSelectCommand(selectSql);

            using (var reader = await selectCommand.ExecuteReaderAsync())
            {
                while (await reader.ReadAsync())
                {
                    var orderId = reader.GetFieldValue<Int64>("OrderId");
                    await UpdateShipStatusAsync(connection, orderId);
                }   
            }       
        }

        Console.WriteLine("Data inserted.");
    }

    private async Task UpdateShipStatusAsync(SpannerConnection connection, Int64 orderId)
    {
        var ordersCommand = connection.CreateUpdateCommand("Orders", 
                new SpannerParameterCollection {
                    { "OrderId", SpannerDbType.Int64, orderId },
                    { "Status", SpannerDbType.String, "SHIPPED" },
                    { "LastUpdateTime", SpannerDbType.Timestamp, SpannerParameter.CommitTimestamp } } );

        var historyCommand = connection.CreateInsertCommand("OrdersHistory", 
                new SpannerParameterCollection {
                    { "OrderId", SpannerDbType.Int64, orderId },
                    { "TimeStamp", SpannerDbType.Timestamp, SpannerParameter.CommitTimestamp } } );

        await connection.RunWithRetriableTransactionAsync(async transaction =>
        {
            ordersCommand.Transaction = transaction;
            await ordersCommand.ExecuteNonQueryAsync();

            historyCommand.Transaction = transaction;
            await historyCommand.ExecuteNonQueryAsync();
        });
    }
}