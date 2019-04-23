import argparse
import redis


def main(args):
    r = redis.Redis(db=7)
    listname = args.listname + 'list'
    ptn = args.ptn
    if args.delete:
        if r.hdel(listname, ptn) > 0:
            print('delete `{}` from {}'.format(ptn, listname))
        else:
            print('Not found: {}: `{}`'.format(listname, ptn))
    else:
        r.hincrby(listname, args.ptn, 0)
        print('added `{}` into {}'.format(ptn, listname))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Import Lists into Redis')
    parser.add_argument(
        '--list', dest='listname', type=str, required=True,
        choices=['black', 'white', 'domain'],
    )
    parser.add_argument('--delete', action='store_true')
    parser.add_argument('ptn', type=str)
    args = parser.parse_args()
    main(args)
