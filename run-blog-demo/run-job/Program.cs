Console.WriteLine("Starting job ...");

var newOrder = new InsertOrderAsync();
await newOrder.InsertDataAsync();

Console.WriteLine("Update done. Job completed.");
