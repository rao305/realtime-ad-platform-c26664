"""Stream ad events into BigQuery for analytics.

A second consumer group on the SAME `ad-events` topic streams raw events into a
partitioned BigQuery table. Serving truth (Postgres) and analytics (BigQuery)
are separate sinks fed by the same log — neither can slow the other.
"""
from kafka import KafkaConsumer
from google.cloud import bigquery
import json

client = bigquery.Client()
TABLE = "adplatform.analytics.events"  # date-partitioned, clustered by campaign


def run() -> None:
    consumer = KafkaConsumer(
        "ad-events",
        bootstrap_servers="kafka:9092",
        group_id="bq-sink",          # independent group → independent offset
        auto_offset_reset="earliest",
    )
    batch: list[dict] = []
    for msg in consumer:
        batch.append(json.loads(msg.value))
        if len(batch) >= 500:        # batch inserts: BigQuery loves bulk, not row-by-row
            client.insert_rows_json(TABLE, batch)
            batch.clear()
