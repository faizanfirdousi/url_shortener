# URL Shortener - DevOps Project

A URL shortener service built with Go, PostgreSQL, and Redis, deployed on AWS EC2 with Docker Compose and automated CI/CD via GitHub Actions.

## 🏗️ Architecture

This project demonstrates a complete DevOps workflow with a microservices architecture:

### Application Layer
- **Go Backend API**: RESTful service handling URL shortening and redirection
- **Frontend**: Single-page application for user interaction
- **Health Endpoints**: Monitoring and service health checks

### Data Layer
- **PostgreSQL**: Persistent storage for URL mappings and metadata
- **Redis**: In-memory cache for frequently accessed URLs, reducing database load

### Infrastructure & Deployment
- **Docker**: Containerized application with multi-stage builds for optimized images
- **Docker Compose**: Orchestrates all services (app, PostgreSQL, Redis) with health checks and dependencies
- **AWS EC2**: Cloud compute instance hosting the containerized application
- **Docker Hub**: Container registry storing built images

### CI/CD Pipeline
- **GitHub Actions CI**: Automatically builds Docker image on code push and pushes to Docker Hub
- **GitHub Actions CD**: Self-hosted runner on EC2 pulls latest image and deploys using Docker Compose
- **Automated Deployment**: Zero-downtime deployments with container orchestration

### Flow
1. Developer pushes code to `main` branch
2. GitHub Actions builds Docker image and pushes to Docker Hub
3. CD pipeline triggers on EC2, pulls latest image
4. Docker Compose orchestrates deployment with health checks
5. Application serves requests with PostgreSQL persistence and Redis caching

## 🚀 Tech Stack

- **Backend**: Go 1.22 (Chi router)
- **Database**: PostgreSQL 15
- **Cache**: Redis 7
- **Frontend**: Vanilla JavaScript
- **Infrastructure**: AWS EC2
- **Containerization**: Docker & Docker Compose
- **CI/CD**: GitHub Actions
- **Registry**: Docker Hub

## 🏃 Quick Start

```bash
git clone <repository-url>
cd url-shortener
docker-compose up -d
```

Access at http://localhost:8082

## 🏭 Infrastructure

- **EC2 Instance**: Amazon Linux 2023 or Ubuntu 22.04
- **Security Group**: Ports 22 (SSH), 80 (HTTP), 443 (HTTPS), 8082 (App)
- **Elastic IP**: Static IP address for the instance
- **Docker & Docker Compose**: Installed on EC2
- **Deployment**: Copy `docker-compose.prod.yml` to EC2, configure `.env` file

## 🔄 CI/CD Pipeline

**GitHub Actions Workflows:**

1. **CI** (`.github/workflows/ci.yml`): Builds and pushes Docker image to Docker Hub on push to `main`
2. **CD** (`.github/workflows/cd.yml`): Deploys to EC2 using self-hosted runner

**Required Secrets:**
- `DOCKER_USERNAME`
- `DOCKER_PASSWORD`

## 🚢 Deployment

1. Configure GitHub Secrets (`DOCKER_USERNAME`, `DOCKER_PASSWORD`)
2. Set up EC2 instance with Docker and GitHub Actions runner
3. Deploy `docker-compose.prod.yml` with `.env` configuration
4. Push to `main` branch to trigger automatic deployment
