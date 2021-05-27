CREATE SCHEMA IF NOT EXISTS flight;
CREATE TABLE IF NOT EXISTS flight ( 
  flight_id INT AUTO_INCREMENT PRIMARY KEY,
  name varchar(100) NOT NULL,
  destination varchar(10) NOT NULL
);

INSERT INTO flight (name, destination) VALUES ('FLIGHT 33', 'VXO'), ('Air Canada Flight 1337', 'LAX'); 
