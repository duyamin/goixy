import argparse
import redis


def get_list(list_name):
    r = redis.Redis(db=7)
    return sorted([x for x in r.hgetall(list_name).items()], key=lambda x: int(x[1]))


def main(args):
    records = get_list(args.listname + 'list')
    for x in records:
        print('{}    {}'.format(x[0].decode('utf-8'), int(x[1])))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Read Lists from Redis')
    parser.add_argument(
        '--list', dest='listname', type=str, required=True,
        choices=['black', 'bytes', 'domain', 'ok', 'white'],
    )
    args = parser.parse_args()
    main(args)
