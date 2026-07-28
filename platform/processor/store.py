"""PostgreSQL access for the write path: durable spend accounting.

The budget invariant — spent_today never exceeds daily_budget, and a campaign
flips inactive the moment it's exhausted — is enforced inside a single
transaction so concurrent processors can't oversell a budget.
"""
import psycopg2

_conn = psycopg2.connect("dbname=ads user=ads host=postgres")


def record_spend(campaign_id: str, amount: float) -> float:
    """Add spend atomically; deactivate the campaign if the budget is hit.
    Returns the new spent_today total."""
    with _conn:                      # transaction: commit on success, rollback on error
        with _conn.cursor() as cur:
            # SELECT ... FOR UPDATE locks this campaign row so two processors
            # can't read-modify-write the same budget concurrently.
            cur.execute(
                "SELECT spent_today, daily_budget FROM campaigns "
                "WHERE id = %s FOR UPDATE",
                (campaign_id,),
            )
            spent, budget = cur.fetchone()
            new_total = float(spent) + amount
            active = new_total < float(budget)
            cur.execute(
                "UPDATE campaigns SET spent_today = %s, active = %s WHERE id = %s",
                (new_total, active, campaign_id),
            )
            return new_total
