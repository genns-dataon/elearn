# AWS Deployment Guide

This guide will help you deploy the AI eLearning Platform to AWS using EC2 and Docker.

## Prerequisites

- AWS Account
- AWS CLI installed and configured
- SSH key pair for EC2 access
- Your OpenAI and/or Anthropic API keys

## Option 1: Simple EC2 Deployment (Recommended for Beginners)

### Step 1: Launch an EC2 Instance

1. **Go to AWS Console** → EC2 → Launch Instance

2. **Configure Instance:**
   - **Name:** elearning-app
   - **AMI:** Amazon Linux 2023 or Ubuntu 22.04 LTS
   - **Instance Type:** t3.medium (2 vCPU, 4GB RAM) minimum
   - **Key Pair:** Create new or select existing
   - **Network Settings:**
     - Allow SSH (port 22) from your IP
     - Allow HTTP (port 80) from anywhere (0.0.0.0/0)
     - Allow HTTPS (port 443) from anywhere (0.0.0.0/0)
     - Allow Custom TCP (port 8080) from anywhere (for API)
   - **Storage:** 30 GB gp3

3. **Launch Instance**

### Step 2: Connect to Your Instance

```bash
# Replace with your key and instance IP
ssh -i your-key.pem ec2-user@your-instance-ip

# Or for Ubuntu:
ssh -i your-key.pem ubuntu@your-instance-ip
```

### Step 3: Install Docker and Docker Compose

**For Amazon Linux 2023:**
```bash
sudo yum update -y
sudo yum install -y docker
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -a -G docker ec2-user

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Log out and back in for group changes
exit
```

**For Ubuntu:**
```bash
sudo apt update
sudo apt install -y docker.io docker-compose
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -a -G docker ubuntu

# Log out and back in
exit
```

### Step 4: Clone and Configure Your App

```bash
# SSH back in
ssh -i your-key.pem ec2-user@your-instance-ip

# Install git if needed
sudo yum install -y git  # Amazon Linux
# OR
sudo apt install -y git  # Ubuntu

# Clone your repository (or upload files)
git clone <your-repo-url>
cd elearn

# Create .env file
cat > .env << 'EOF'
MODEL_PROVIDER=openai
ANTHROPIC_MODEL=claude-3-5-sonnet-20241022
ANTHROPIC_API_KEY=your_anthropic_key_here
OPENAI_MODEL=gpt-4o-mini
OPENAI_API_KEY=your_openai_key_here
EMBEDDING_PROVIDER=openai
EMBEDDING_MODEL=text-embedding-3-small
OLLAMA_HOST=http://localhost:11434
EOF

# Set your actual API keys
nano .env  # Edit the file with your keys
```

### Step 5: Update Frontend API URL

Before building, update the nginx configuration to proxy API requests:

```bash
# Edit web/nginx.conf
cat > web/nginx.conf << 'EOF'
server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    # Proxy API requests to backend
    location /api/ {
        proxy_pass http://api:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Proxy audio files
    location /audio/ {
        proxy_pass http://api:8080;
        proxy_http_version 1.1;
    }

    # SPA routing
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Cache static assets
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
}
EOF
```

### Step 6: Build and Run with Docker Compose

```bash
# Build and start containers
docker-compose up -d --build

# Check status
docker-compose ps

# View logs
docker-compose logs -f

# To stop
docker-compose down

# To restart after changes
docker-compose down && docker-compose up -d --build
```

### Step 7: Access Your Application

Open your browser and navigate to:
- **Frontend:** http://your-instance-ip
- **API Health:** http://your-instance-ip/api/health

### Step 8: Set Up Domain (Optional)

1. **Get a domain** from Route 53 or external registrar
2. **Point DNS A record** to your EC2 instance IP
3. **Install SSL certificate** with Let's Encrypt:

```bash
# Install certbot
sudo yum install -y certbot python3-certbot-nginx  # Amazon Linux
# OR
sudo apt install -y certbot python3-certbot-nginx  # Ubuntu

# Get certificate (replace yourdomain.com)
sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com

# Auto-renew setup
sudo certbot renew --dry-run
```

## Option 2: AWS Elastic Beanstalk (Easier Management)

1. **Install EB CLI:**
```bash
pip install awsebcli
```

2. **Initialize Elastic Beanstalk:**
```bash
cd elearn
eb init -p docker elearning-app --region us-east-1
```

3. **Create environment:**
```bash
eb create elearning-prod
```

4. **Set environment variables:**
```bash
eb setenv \
  MODEL_PROVIDER=openai \
  OPENAI_API_KEY=your_key \
  OPENAI_MODEL=gpt-4o-mini \
  EMBEDDING_PROVIDER=openai \
  EMBEDDING_MODEL=text-embedding-3-small
```

5. **Deploy:**
```bash
eb deploy
```

6. **Open app:**
```bash
eb open
```

## Option 3: AWS ECS with Fargate (Serverless Containers)

This is more advanced but provides better scaling:

1. **Create ECR repositories** for your images
2. **Build and push images** to ECR
3. **Create ECS cluster** with Fargate
4. **Create task definitions** for API and Web
5. **Create ALB** (Application Load Balancer)
6. **Deploy services**

See AWS ECS documentation for detailed steps.

## Monitoring and Maintenance

### View Logs
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f api
docker-compose logs -f web
```

### Restart Services
```bash
docker-compose restart api
docker-compose restart web
```

### Update Application
```bash
# Pull latest code
git pull

# Rebuild and restart
docker-compose down
docker-compose up -d --build
```

### Backup Database
```bash
# Copy database from container
docker-compose exec api cp /app/storage/elearn.db /app/storage/backup.db

# Download from EC2
scp -i your-key.pem ec2-user@your-instance-ip:/path/to/backup.db ./local-backup.db
```

### Monitor Resources
```bash
# Container stats
docker stats

# Disk usage
df -h

# Memory usage
free -m
```

## Troubleshooting

### Containers won't start
```bash
docker-compose logs
docker-compose ps
```

### Port already in use
```bash
# Find process using port 80 or 8080
sudo lsof -i :80
sudo lsof -i :8080

# Kill process if needed
sudo kill -9 <PID>
```

### Out of disk space
```bash
# Clean up Docker
docker system prune -a

# Remove old images
docker image prune -a
```

### API can't connect
```bash
# Check if API is running
curl http://localhost:8080/api/health

# Check network
docker network ls
docker network inspect elearn_default
```

## Security Best Practices

1. **Use HTTPS** in production (Let's Encrypt)
2. **Restrict SSH access** to your IP only
3. **Use AWS Secrets Manager** for API keys
4. **Enable CloudWatch** for monitoring
5. **Regular security updates:**
```bash
sudo yum update -y  # Amazon Linux
sudo apt update && sudo apt upgrade -y  # Ubuntu
```

6. **Set up CloudWatch alarms** for:
   - High CPU usage
   - High memory usage
   - Disk space usage
   - HTTP error rates

## Cost Optimization

- **Use t3.medium** for production (~$30/month)
- **Use t3.small** for testing (~$15/month)
- **Enable auto-shutdown** for non-production instances
- **Use Reserved Instances** for long-term savings
- **Monitor costs** with AWS Cost Explorer

## Support

For issues:
1. Check logs: `docker-compose logs -f`
2. Check health: `curl http://localhost:8080/api/health`
3. Restart services: `docker-compose restart`
4. Review this guide's troubleshooting section
