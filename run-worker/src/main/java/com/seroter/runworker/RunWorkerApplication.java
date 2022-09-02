package com.seroter.runworker;


import java.util.function.Consumer;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;
import reactor.core.publisher.Flux;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.Statement;
import java.sql.SQLException;


@SpringBootApplication
public class RunWorkerApplication {

	public static void main(String[] args) {
		SpringApplication.run(RunWorkerApplication.class, args);
	}

	//takes in a Flux (stream) of orders
	@Bean
	public Consumer<Flux<Order>> reactiveReadOrders() {

		//connection to my database
		String connectionUrl = String.format("jdbc:cloudspanner:/%s", System.getenv("SPANNER_URI"));
		
		return value -> 
			value.subscribe(v -> { 
				try (Connection c = DriverManager.getConnection(connectionUrl); Statement statement = c.createStatement()) {
					String command = "UPDATE Orders SET Status = 'DELIVERED', LastUpdateTime = PENDING_COMMIT_TIMESTAMP() WHERE OrderId = " + v.getOrderId().toString();
					statement.executeUpdate(command);
				} catch (SQLException e) {
					System.out.println(e.toString());
				}
			});
	}
}
