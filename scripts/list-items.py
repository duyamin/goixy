import argparse
import redis


def get_list(list_name):
    r = redis.Redis(db=7)
    return sorted([x for x in r.hgetall(list_name).items()], key=lambda x: int(x[1]))


def main(args):
    records = get_list(args.listname + 'list')
    for x in records:
        if args.ptn_only:
            print('{}'.format(x[0].decode('utf-8')))
        else:
            print('{}    {}'.format(x[0].decode('utf-8'), int(x[1])))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Read Lists from Redis')
    parser.add_argument(
        'listname', type=str,
        choices=['black', 'bytes', 'domain', 'ok', 'white'],
    )
    parser.add_argument(
        '-o', '-n', '--ptn', dest='ptn_only', action='store_true',
        help='ptn only, without numbers'
    )
    args = parser.parse_args()
    main(args)
