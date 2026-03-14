import time
import requests


class TradeEngineClient:
    """Python client for the TradeEngine Gateway API."""

    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url.rstrip("/")
        self.session = requests.Session()

    def create_order(self, user_id, symbol, side, quantity, price, idempotency_key=None):
        """Submit a new order. Returns the order dict with status 'pending'."""
        payload = {
            "user_id": user_id,
            "symbol": symbol,
            "side": side,
            "quantity": quantity,
            "price": str(price),
        }
        if idempotency_key:
            payload["idempotency_key"] = idempotency_key
        resp = self.session.post(f"{self.base_url}/orders", json=payload)
        resp.raise_for_status()
        return resp.json()

    def get_order(self, order_id):
        """Fetch a single order by ID."""
        resp = self.session.get(f"{self.base_url}/orders/{order_id}")
        resp.raise_for_status()
        return resp.json()

    def list_orders(self, user_id):
        """List all orders for a user."""
        resp = self.session.get(f"{self.base_url}/orders", params={"user_id": user_id})
        resp.raise_for_status()
        return resp.json()

    def get_positions(self, user_id):
        """List all positions for a user."""
        resp = self.session.get(f"{self.base_url}/positions", params={"user_id": user_id})
        resp.raise_for_status()
        return resp.json()

    def get_user(self, user_id):
        """Fetch a user by ID."""
        resp = self.session.get(f"{self.base_url}/users/{user_id}")
        resp.raise_for_status()
        return resp.json()

    def wait_for_order(self, order_id, timeout=10, poll_interval=0.5):
        """Poll until the order reaches a terminal status (executed, settled, rejected)."""
        terminal = {"executed", "settled", "rejected"}
        deadline = time.time() + timeout
        while time.time() < deadline:
            order = self.get_order(order_id)
            if order.get("status") in terminal:
                return order
            time.sleep(poll_interval)
        raise TimeoutError(f"order {order_id} did not reach terminal status within {timeout}s")
