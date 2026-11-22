# Account and User Management Setup

## Generate Operator NKey

The account and user management feature requires an operator NKey to sign account JWTs.

### Option 1: Using nk tool

```bash
# Install nk tool
go install github.com/nats-io/nkeys/nk@latest

# Generate operator keypair
nk -gen operator -pubout
```

This will output two keys:
```
SOABC123...  # Seed (private key) - use this for OPERATOR_NKEY_SEED
OABC123...   # Public key
```

### Option 2: Using nsc tool

```bash
# Install nsc
brew tap nats-io/nats-tools
brew install nsc

# Create operator
nsc add operator MyOperator
```

## Configure Application

### For Docker Compose

Edit `docker-compose.yml`:

```yaml
backend:
  environment:
    - OPERATOR_NKEY_SEED=SOABC123...  # Your operator seed here
```

Then restart:
```bash
docker-compose down
docker-compose up --build
```

### For Local Development

Set environment variable:

```bash
# Bash/Zsh
export OPERATOR_NKEY_SEED="SOABC123..."

# Or create .env file
echo 'OPERATOR_NKEY_SEED=SOABC123...' > backend/.env
```

Then run:
```bash
make backend  # In one terminal
make frontend # In another terminal
```

## Verify Setup

1. Start the application
2. Check backend logs for:
   ```
   Account and user management enabled
   ```

3. Navigate to http://localhost:3000/accounts
4. You should be able to create accounts and users

## Without Operator Key

If you don't set `OPERATOR_NKEY_SEED`:
- Application will start normally
- Account/user endpoints will be disabled  
- You'll see: `WARNING: OPERATOR_NKEY_SEED not set - account/user management will not work`
- Other features (streams, consumers, messages) work normally

## Security Notes

⚠️ **Keep your operator seed secret!**
- Never commit it to version control
- Use environment variables or secure secret management
- Anyone with the operator seed can create accounts in your system
