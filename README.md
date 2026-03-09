# sheepy-wallet

HD-кошелёк для Ethereum. Генерация адресов (BIP-44) и offline-подпись EIP-1559 транзакций.

## Запуск

### Docker
```bash
docker compose up wallet
```

### Локально
```bash
cp config.example.json config.json
# отредактировать мнемонику в config.json
go build -o sheepy-wallet ./cmd/server/
./sheepy-wallet
```

## API

### POST /api/v1/createaddress
```bash
curl -s -X POST http://127.0.0.1:8000/api/v1/createaddress \
  -H "Content-Type: application/json" \
  -d '{"gate":"ethereum","account":0,"change":0,"address_index":0}'
```

### POST /api/v1/validateaddress
```bash
curl -s -X POST http://127.0.0.1:8000/api/v1/validateaddress \
  -H "Content-Type: application/json" \
  -d '{"address":"0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"}'
```

### POST /api/v1/tx
```bash
curl -s -X POST http://127.0.0.1:8000/api/v1/tx \
  -H "Content-Type: application/json" \
  -d '{
    "gate":"ethereum","account":0,"change":0,"address_index":0,
    "tx_params":{
      "to":"0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
      "value_wei":"1000000000000000000",
      "data":"0x","nonce":0,"chain_id":11155111,
      "gas_limit":21000,
      "max_fee_per_gas_wei":"30000000000",
      "max_priority_fee_per_gas_wei":"1000000000"
    }
  }'
```

Значения `value_wei`, `max_fee_per_gas_wei`, `max_priority_fee_per_gas_wei` — десятичные строки, не hex.

## Конфигурация

`config.json` (в `.gitignore`, в репо только `config.example.json`):

```json
{
  "config": { "host": "127.0.0.1", "port": 8000 },
  "gates": [{ "name": "ethereum", "mnemonic": "your mnemonic phrase here" }]
}
```

Поля: `host`, `port`, `gates[].name`, `gates[].mnemonic`.
