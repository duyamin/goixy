import argparse
import redis


def main(args):
    r = redis.Redis(db=7)
    listname = args.listname + 'list'
    with open(args.filename) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            line = line.split(' ')[0]
            r.hincrby(listname, line, 0)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Import Lists into Redis')
    parser.add_argument(
        '-f', dest='filename', type=str, required=True,
    )
    parser.add_argument(
        '--list', dest='listname', type=str, required=True,
        choices=['black', 'white', 'domain'],
    )
    args = parser.parse_args()
    main(args)
