#!/bin/sh

if [ -z "$1" ]; then
	echo "usage: $0 version"
	exit 1
fi

gem=`whereis gem | head -n 1 | awk '{ print $2 }'`

if [ -z "$gem" ] || [ ! -x "$gem" ]; then
	echo "ruby gems not found, assuming fpm isn't installed"
	exit 1
fi

ruby=`head -n 1 "$gem" | sed 's/^#!//' | grep ruby`
if [ -z "$ruby" ]; then
       	echo "$gem isn't ruby gems, assuming fpm isn't installed"
	exit 1
fi

fpm=`find /var/lib/gems/ /usr/lib*/ruby/gems/ -path "*/gems/fpm-*/bin/fpm" 2> /dev/null | sort -r | head -n 1`
if [ -z "$fpm" ]; then
	echo "didn't find fpm, assuming it isn't installed"
fi

rm -f *.deb *.rpm

distributor=`lsb_release -i | awk '{ print $NF }'`
if [ -z "$distributor" ]; then
	echo "lsb_release -i failed to give distro identifier"
	exit 1
elif [ x"$distributor" = x"Ubuntu" -o x"$distributor" = x"Debian" ]; then
        package_type="deb"
	init_script="./packaging/upstart=/etc/init/restatemachine.conf"
	postinit="./packaging/upstart_postinit.sh"
elif [ x"$distributor" = x"RedHatEnterpriseServer" ]; then
	package_type="rpm"
	init_script="./packaging/upstart=/etc/init/restatemachine.conf"
	postinit="./packaging/upstart_postinit.sh"
fi

if [ ! -x ./restatemachine ]; then
	echo "no built binary found in PWD, you should execute this script as:"
	echo "make package"
	exit 1
fi

$fpm -s dir -t "$package_type" -n restatemachine --config-files /etc/restatemachine/restatemachine.conf -v "$1" --after-install "$postinit" \
	--description "Restatemachine is generic state machine executor available through a RESTful API" \
	-m "Atomia AB <info@atomia.com>" --vendor "Atomia AB" --url http://github.com/atomia/restatemachine \
	./restatemachine=/usr/bin/restatemachine ./default_config/=/etc/restatemachine "$init_script"
