-- +goose Up
CREATE TABLE packets (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    time VARCHAR(255) NOT NULL,
    latitude DOUBLE NOT NULL,
    longitude DOUBLE NOT NULL,
    satellites INT NOT NULL,
    acceleration_x DOUBLE NOT NULL,
    acceleration_y DOUBLE NOT NULL,
    acceleration_z DOUBLE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_time (time),
    INDEX idx_coordinates (latitude, longitude),
    INDEX idx_created_at (created_at)
);

-- +goose Down
DROP TABLE packets;
