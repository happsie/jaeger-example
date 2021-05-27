CREATE SCHEMA IF NOT EXISTS passenger; 
CREATE TABLE IF NOT EXISTS passenger ( 
  passenger_id INT AUTO_INCREMENT PRIMARY KEY,
  flight_id INT NOT NULL,
  firstname varchar(100) NOT NULL,
  surname varchar(100) NOT NULL
);

INSERT INTO passenger (flight_id, firstname, surname) VALUES (1, 'Jesper', 'Placeholdersson'), (2, 'Olof', 'Coolsson'), (2, 'Elliot', 'Vimsson'); 
